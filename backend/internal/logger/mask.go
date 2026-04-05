package logger

import "strings"

// MaskEmail masks an email address for safe logging.
// Example: "kyle@gmail.com" -> "k***@gmail.com"
func MaskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "***"
	}
	return string(parts[0][0]) + "***@" + parts[1]
}

// MaskToken masks a token string, keeping only the first 8 characters.
// Example: "eyJhbGciOiJIUzI1NiJ9..." -> "eyJhbGci..."
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:8] + "..."
}
