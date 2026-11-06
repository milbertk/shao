package emailnotifier

import (
	"crypto/rand"
	"encoding/hex"
)

// NewPublicToken returns a random, unguessable token for the customer
// "manage my request" links. The prefix tells which record type it is:
// "a" = appointment, "w" = web order, "o" = order (e.g. "a_9f3c...").
func NewPublicToken(prefix string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(buf), nil
}
