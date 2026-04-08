package llm

import (
	"errors"
	"net"
	"testing"
)

// withLookupIP replaces the package-level lookupIP hook for the duration of
// a test so SSRF unit tests never touch real DNS. The returned closure MUST
// be deferred to restore the production default.
func withLookupIP(t *testing.T, fn func(host string) ([]net.IP, error)) func() {
	t.Helper()
	prev := lookupIP
	lookupIP = fn
	return func() { lookupIP = prev }
}

func TestValidateBaseURL_AllowedHTTPS(t *testing.T) {
	// Public anthropic endpoint resolves to a global IP; stub the lookup so
	// the test stays hermetic and never depends on the network.
	defer withLookupIP(t, func(_ string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("104.18.32.47")}, nil
	})()

	if err := ValidateBaseURL("https://api.anthropic.com/v1/messages"); err != nil {
		t.Fatalf("expected nil for allowed https host, got %v", err)
	}
}

func TestValidateBaseURL_RejectsNonHTTPS(t *testing.T) {
	// No DNS lookup should happen for non-https; sentinel the hook to fail
	// the test if it's ever called.
	defer withLookupIP(t, func(_ string) ([]net.IP, error) {
		t.Fatal("lookupIP must not be called for non-https scheme")
		return nil, nil
	})()

	cases := []string{
		"http://example.com",
		"ftp://example.com",
		"file:///etc/passwd",
		"gopher://example.com",
		"ws://example.com",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			err := ValidateBaseURL(raw)
			if !errors.Is(err, ErrSSRFBadScheme) {
				t.Fatalf("%s: expected ErrSSRFBadScheme, got %v", raw, err)
			}
		})
	}
}

func TestValidateBaseURL_RejectsLocalhostByHostname(t *testing.T) {
	defer withLookupIP(t, func(_ string) ([]net.IP, error) {
		t.Fatal("lookupIP must not be called when hostname is blocked")
		return nil, nil
	})()

	err := ValidateBaseURL("https://localhost/api")
	if !errors.Is(err, ErrSSRFHostBlocked) {
		t.Fatalf("expected ErrSSRFHostBlocked for localhost, got %v", err)
	}
}

func TestValidateBaseURL_RejectsMetadataHostname(t *testing.T) {
	defer withLookupIP(t, func(_ string) ([]net.IP, error) {
		t.Fatal("lookupIP must not be called when metadata host is blocked")
		return nil, nil
	})()

	cases := []string{
		"https://metadata.google.internal/compute",
		"https://169.254.169.254/latest/meta-data/",
		"https://metadata/",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			err := ValidateBaseURL(raw)
			if !errors.Is(err, ErrSSRFMetadataHost) {
				t.Fatalf("%s: expected ErrSSRFMetadataHost, got %v", raw, err)
			}
		})
	}
}

func TestValidateBaseURL_RejectsDotLocalSuffix(t *testing.T) {
	defer withLookupIP(t, func(_ string) ([]net.IP, error) {
		t.Fatal("lookupIP must not be called when .local suffix is blocked")
		return nil, nil
	})()

	err := ValidateBaseURL("https://example.local/api")
	if !errors.Is(err, ErrSSRFHostBlocked) {
		t.Fatalf("expected ErrSSRFHostBlocked for .local, got %v", err)
	}
}

func TestValidateBaseURL_RejectsPrivateIPv4(t *testing.T) {
	cases := []struct {
		raw string
		ip  net.IP
	}{
		{"https://host.example.com/", net.ParseIP("10.0.0.1")},
		{"https://host.example.com/", net.ParseIP("172.16.5.2")},
		{"https://host.example.com/", net.ParseIP("192.168.1.1")},
		{"https://host.example.com/", net.ParseIP("127.0.0.1")},
		{"https://host.example.com/", net.ParseIP("169.254.1.1")},
	}
	for _, tc := range cases {
		t.Run(tc.ip.String(), func(t *testing.T) {
			defer withLookupIP(t, func(_ string) ([]net.IP, error) {
				return []net.IP{tc.ip}, nil
			})()
			err := ValidateBaseURL(tc.raw)
			if !errors.Is(err, ErrSSRFPrivateIP) {
				t.Fatalf("ip %s: expected ErrSSRFPrivateIP, got %v", tc.ip, err)
			}
		})
	}
}

func TestValidateBaseURL_RejectsPrivateIPv6(t *testing.T) {
	cases := []net.IP{
		net.ParseIP("::1"),
		net.ParseIP("fc00::1"),
		net.ParseIP("fe80::1"),
	}
	for _, ip := range cases {
		t.Run(ip.String(), func(t *testing.T) {
			defer withLookupIP(t, func(_ string) ([]net.IP, error) {
				return []net.IP{ip}, nil
			})()
			err := ValidateBaseURL("https://host.example.com/")
			if !errors.Is(err, ErrSSRFPrivateIP) {
				t.Fatalf("ip %s: expected ErrSSRFPrivateIP, got %v", ip, err)
			}
		})
	}
}

func TestValidateBaseURL_RejectsDirectPrivateIPLiteral(t *testing.T) {
	// When the user types a literal private IP, Go's net.LookupIP on most
	// platforms returns the IP as-is, so the resolution path catches it. Stub
	// that explicit behavior to keep the test hermetic.
	defer withLookupIP(t, func(host string) ([]net.IP, error) {
		if ip := net.ParseIP(host); ip != nil {
			return []net.IP{ip}, nil
		}
		return nil, errors.New("unexpected host")
	})()

	err := ValidateBaseURL("https://10.0.0.1/api")
	if !errors.Is(err, ErrSSRFPrivateIP) {
		t.Fatalf("expected ErrSSRFPrivateIP for literal 10.0.0.1, got %v", err)
	}
}

func TestValidateBaseURL_RejectsDNSResolveFailure(t *testing.T) {
	defer withLookupIP(t, func(_ string) ([]net.IP, error) {
		return nil, errors.New("nxdomain")
	})()

	err := ValidateBaseURL("https://nonexistent.example.com/")
	if !errors.Is(err, ErrSSRFHostBlocked) {
		t.Fatalf("expected ErrSSRFHostBlocked on DNS failure, got %v", err)
	}
}

func TestValidateBaseURL_RejectsEmptyHost(t *testing.T) {
	defer withLookupIP(t, func(_ string) ([]net.IP, error) {
		t.Fatal("lookupIP must not be called for empty host")
		return nil, nil
	})()

	err := ValidateBaseURL("https:///api")
	if !errors.Is(err, ErrSSRFHostBlocked) {
		t.Fatalf("expected ErrSSRFHostBlocked for empty host, got %v", err)
	}
}

func TestValidateBaseURL_RejectsMalformedURL(t *testing.T) {
	defer withLookupIP(t, func(_ string) ([]net.IP, error) {
		t.Fatal("lookupIP must not be called for malformed url")
		return nil, nil
	})()

	err := ValidateBaseURL("://bad")
	if !errors.Is(err, ErrSSRFBadScheme) {
		t.Fatalf("expected ErrSSRFBadScheme for malformed url, got %v", err)
	}
}

func TestValidateBaseURL_IPv6DocumentationRange(t *testing.T) {
	// 2001:db8::/32 is the RFC 3849 documentation range. It is not in any of
	// our private blocklists, so a stubbed resolution to a documentation IP
	// must pass. This documents the intentional boundary: we block link-local,
	// ULA, loopback, and IPv4 RFC1918, but public IPv6 routable addresses
	// are allowed.
	defer withLookupIP(t, func(_ string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("2001:db8::1")}, nil
	})()

	err := ValidateBaseURL("https://ipv6.example.com/api")
	if err != nil {
		t.Fatalf("expected nil for documentation IPv6, got %v", err)
	}
}
