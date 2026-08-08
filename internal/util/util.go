// Package util holds small shared helpers used across apps.
package util

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a name to a URL-friendly slug, e.g. "Duka Kuu" -> "duka-kuu".
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// RandomDigits returns a digits-only string of length n (for OTP codes).
func RandomDigits(n int) (string, error) {
	max := big.NewInt(10)
	var b strings.Builder
	for i := 0; i < n; i++ {
		d, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b.WriteByte('0' + byte(d.Int64()))
	}
	return b.String(), nil
}

// RandomToken returns a hex token of n random bytes (for verification links).
func RandomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.ToLower(hex(buf)), nil
}

func hex(b []byte) string {
	const digits = "0123456789abcdef"
	var sb strings.Builder
	for _, c := range b {
		sb.WriteByte(digits[c>>4])
		sb.WriteByte(digits[c&0x0f])
	}
	return sb.String()
}

// Truncate shortens a string to n runes.
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
