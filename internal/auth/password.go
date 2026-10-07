package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	scheme  = "pbkdf2_sha256"
	saltLen = 16
	keyLen  = 32
)

var b64 = base64.RawStdEncoding

// HashPassword derives a key from the password with PBKDF2-HMAC-SHA256 and a
// random per-user salt. Output format (self-describing, so you can raise the
// iteration count later without breaking old hashes):
//
//	pbkdf2_sha256$<iterations>$<salt b64>$<key b64>
func HashPassword(password string, iterations int) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := pbkdf2.Key([]byte(password), salt, iterations, keyLen, sha256.New)
	return fmt.Sprintf("%s$%d$%s$%s", scheme, iterations, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword re-derives the key using the stored salt/iterations and
// compares in constant time.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != scheme {
		return false, errors.New("invalid password hash format")
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false, errors.New("invalid iteration count")
	}
	salt, err := b64.DecodeString(parts[2])
	if err != nil {
		return false, err
	}
	want, err := b64.DecodeString(parts[3])
	if err != nil {
		return false, err
	}
	got := pbkdf2.Key([]byte(password), salt, iterations, len(want), sha256.New)
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
