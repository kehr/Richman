package llm

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// SSRF sentinel errors. Callers MUST classify failures with errors.Is so the
// API layer can map each category to a stable error code in the HTTP
// response without leaking the resolved IP or DNS internals.
var (
	// ErrSSRFBadScheme rejects anything that is not https. http, file, ftp,
	// gopher and ws are all blocked because the LLM HTTP client should only
	// ever talk TLS to a remote provider.
	ErrSSRFBadScheme = errors.New("llm: base_url must use https scheme")
	// ErrSSRFHostBlocked covers empty hosts, the explicit blocked-hostname
	// list, .local suffix, and DNS resolution failures. The last case is
	// deliberately conflated with "blocked" so attackers cannot probe which
	// of their inputs are just unreachable versus actively banned.
	ErrSSRFHostBlocked = errors.New("llm: base_url hostname blocked")
	// ErrSSRFPrivateIP is returned when ANY resolved A/AAAA record falls
	// inside the private/link-local CIDRs below. Even one private hit
	// blocks the whole hostname to defeat mixed-IP rebinding.
	ErrSSRFPrivateIP = errors.New("llm: base_url resolves to private IP range")
	// ErrSSRFMetadataHost flags the cloud metadata endpoints. This is a
	// subset of the hostname blocklist but surfaces as its own error class
	// so infra logs can alert on likely exfiltration attempts.
	ErrSSRFMetadataHost = errors.New("llm: base_url is a cloud metadata endpoint")
)

// blockedHosts is the exact-match hostname denylist. Values are compared
// case-insensitively after url.Hostname stripping.
var blockedHosts = map[string]bool{
	"localhost":                true,
	"metadata.google.internal": true,
	"169.254.169.254":          true,
	"metadata":                 true,
}

// metadataHosts is the subset of blockedHosts that must surface as
// ErrSSRFMetadataHost instead of the generic ErrSSRFHostBlocked.
var metadataHosts = map[string]bool{
	"metadata.google.internal": true,
	"169.254.169.254":          true,
	"metadata":                 true,
}

// privateCIDRs is the union of IPv4 RFC1918, loopback, link-local, IPv6 ULA
// and IPv6 loopback ranges. Populated once at init time to avoid reparsing
// on every call.
var privateCIDRs []*net.IPNet

// lookupIP is a package-level indirection around net.LookupIP so SSRF unit
// tests can substitute a hermetic DNS resolver. Production callers MUST use
// ValidateBaseURL rather than this variable directly.
var lookupIP = net.LookupIP

func init() {
	cidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"fc00::/7",
		"::1/128",
		"fe80::/10",
	}
	for _, cidr := range cidrs {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			// These CIDRs are compile-time constants; a parse failure would
			// indicate a typo caught by unit tests, not a runtime condition.
			continue
		}
		privateCIDRs = append(privateCIDRs, block)
	}
}

// ValidateBaseURL enforces the SSRF policy documented in the PRD and TRD:
// https-only scheme, hostname denylist, .local suffix block, and DNS-resolved
// private-IP rejection. MUST be called on save (settings handler), on probe,
// and again immediately before every live ChatCompletion call to defeat DNS
// rebinding attacks where the cached hostname is later flipped to an
// internal IP.
//
// The caller should treat ANY non-nil error as "refuse to contact this URL"
// and report it to the user without leaking the resolved IP addresses.
func ValidateBaseURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ErrSSRFBadScheme
	}
	if u.Scheme != "https" {
		return ErrSSRFBadScheme
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return ErrSSRFHostBlocked
	}
	if strings.HasSuffix(host, ".local") {
		return ErrSSRFHostBlocked
	}
	if metadataHosts[host] {
		return ErrSSRFMetadataHost
	}
	if blockedHosts[host] {
		return ErrSSRFHostBlocked
	}
	// DNS resolve and check every A/AAAA record against the private CIDR
	// union. Any miss -> treat as blocked rather than leaking NXDOMAIN vs
	// timeout details.
	ips, err := lookupIP(host)
	if err != nil || len(ips) == 0 {
		return ErrSSRFHostBlocked
	}
	for _, ip := range ips {
		for _, block := range privateCIDRs {
			if block.Contains(ip) {
				return ErrSSRFPrivateIP
			}
		}
	}
	return nil
}

// ValidateSelfHostedBaseURL is the relaxed SSRF policy for user-configured
// self-hosted providers (openai_compatible). It permits both http and https
// schemes and allows loopback / private-network addresses so users can point
// to a local Ollama or an on-premises inference server. Cloud metadata
// endpoints are still blocked because they are never a legitimate LLM target.
func ValidateSelfHostedBaseURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ErrSSRFBadScheme
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return ErrSSRFBadScheme
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return ErrSSRFHostBlocked
	}
	if metadataHosts[host] {
		return ErrSSRFMetadataHost
	}
	return nil
}
