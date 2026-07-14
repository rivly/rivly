package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Cipher struct {
	gcm cipher.AEAD
}

func LoadOrCreate(dir string) (*Cipher, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dir, "secret.key")

	key, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		key = make([]byte, 32)
		if _, gerr := rand.Read(key); gerr != nil {
			return nil, fmt.Errorf("generate key: %w", gerr)
		}
		if werr := os.WriteFile(path, key, 0o600); werr != nil {
			return nil, fmt.Errorf("write key: %w", werr)
		}
	} else if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("secret key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{gcm: gcm}, nil
}

func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (c *Cipher) Decrypt(data []byte) ([]byte, error) {
	nonceSize := c.gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return c.gcm.Open(nil, nonce, ciphertext, nil)
}
