package source

import (
	"crypto/rand"
	"encoding/hex"
)

// RandomID returns a 16-char hex id from crypto/rand. Used for event ids and
// session tokens.
func RandomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
