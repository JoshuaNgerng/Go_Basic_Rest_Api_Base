// Package validate holds the input rules shared by several features, so the
// same email/name/password rules apply at registration, profile update and
// password change.
package validate

import (
	"net/mail"
	"strings"
)

// Error means the caller sent unacceptable input. Handlers map it to a 400.
type Error struct{ Message string }

func (e *Error) Error() string { return e.Message }

// Email trims, lower-cases and checks an email, returning the normalized form.
// Login lower-cases emails, so every stored email must be stored that way.
func Email(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return "", &Error{"invalid email"}
	}
	return s, nil
}

func Name(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", &Error{"name is required"}
	}
	return s, nil
}

func Password(s string) error {
	if len(s) < 8 || len(s) > 128 {
		return &Error{"password must be 8-128 characters"}
	}
	return nil
}
