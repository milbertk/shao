package emailnotifier

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Secrets saved in the database (for example a company's SMTP password) are
// encrypted with AES-256-GCM. The key comes from the environment:
//
//   - SMTP_ENCRYPTION_KEY (recommended; any long random text), or
//   - JWT_SECRET as a fallback, so it works with the current .env.
//
// IMPORTANT: if the key changes, saved SMTP passwords can no longer be read
// and each company must type its SMTP password again.
func secretKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("SMTP_ENCRYPTION_KEY"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	}
	if raw == "" {
		return nil, errors.New("no encryption key: set SMTP_ENCRYPTION_KEY (or JWT_SECRET) in the environment")
	}
	sum := sha256.Sum256([]byte("shao-secret-box:" + raw))
	return sum[:], nil
}

// EncryptSecret encrypts a text and returns "v1:" + base64(nonce|ciphertext).
func EncryptSecret(plain string) (string, error) {
	key, err := secretKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return "v1:" + base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptSecret reverses EncryptSecret.
func DecryptSecret(stored string) (string, error) {
	if !strings.HasPrefix(stored, "v1:") {
		return "", errors.New("unknown secret format")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, "v1:"))
	if err != nil {
		return "", fmt.Errorf("invalid secret encoding: %w", err)
	}
	key, err := secretKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("invalid secret")
	}
	plain, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("the secret could not be decrypted (was the encryption key changed?)")
	}
	return string(plain), nil
}
