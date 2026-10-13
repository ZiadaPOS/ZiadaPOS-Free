package accounts

import (
	"errors"
	"strings"
)

// isUniqueViolation detects Postgres 23505 unique-constraint errors (which
// GORM surfaces wrapped).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "unique constraint") || strings.Contains(msg, "duplicate key")
}

var _ = errors.Is
