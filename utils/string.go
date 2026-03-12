package utils

import (
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
func IsValidUUID(u string) bool {
	if len(u) != 36 {
		return false
	}
	for i, r := range u {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// ContainsAny checks if string contains any of substrings
func ContainsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
