package utils

import (
	"regexp"
	"strings"
)

// SanitizeURL removes whitespace from URL
func SanitizeURL(s string) string {
	return strings.Join(strings.Fields(s), "")
}

// TruncateString truncates string to max runes
func TruncateString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}

// CountRunes counts runes in string
func CountRunes(s string) int {
	return len([]rune(s))
}

// IsValidUUID checks if string is valid UUID
func IsValidUUID(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))

	if len(s) == 38 && s[0] == '{' && s[37] == '}' {
		s = s[1:37]
	}

	if strings.HasPrefix(s, "urn:uuid:") {
		s = s[9:]
	}

	if len(s) != 36 {
		return false
	}

	const uuidPattern = `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	matched, _ := regexp.MatchString(uuidPattern, s)
	return matched
}

// ContainsAnySubstring checks if string contains any of substrings
func ContainsAnySubstring(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
