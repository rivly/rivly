package secret

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateGeneratesAProtectedKey(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")

	if _, err := LoadOrCreate(dir); err != nil {
		t.Fatalf("LoadOrCreate: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("data directory mode = %o, want 700", perm)
	}

	path := filepath.Join(dir, "secret.key")
	info, err = os.Stat(path)
	if err != nil {
		t.Fatalf("Stat key: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("key mode = %o, want 600, the key must never be group or world readable", perm)
	}

	key, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("key length = %d, want 32", len(key))
	}
	if bytes.Equal(key, make([]byte, 32)) {
		t.Fatal("the key must be random, not zeroed")
	}
}

func TestLoadOrCreateKeepsTheExistingKey(t *testing.T) {
	dir := t.TempDir()

	first, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("first LoadOrCreate: %v", err)
	}
	key, err := os.ReadFile(filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	second, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("second LoadOrCreate: %v", err)
	}
	again, err := os.ReadFile(filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !bytes.Equal(key, again) {
		t.Fatal("restarting must not rotate the key, every stored credential would become unreadable")
	}

	sealed, err := first.Encrypt([]byte("ghp_token"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	opened, err := second.Decrypt(sealed)
	if err != nil {
		t.Fatalf("Decrypt across restarts: %v", err)
	}
	if string(opened) != "ghp_token" {
		t.Fatalf("round trip = %q", opened)
	}
}

func TestLoadOrCreateRejectsAKeyOfTheWrongLength(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "secret.key"), []byte("too short"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := LoadOrCreate(dir); err == nil {
		t.Fatal("a truncated key file must abort startup rather than be padded or replaced")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	cipher := newCipher(t)

	for _, plaintext := range []string{"", "a", "correct horse battery staple", string(make([]byte, 4096))} {
		sealed, err := cipher.Encrypt([]byte(plaintext))
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		opened, err := cipher.Decrypt(sealed)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if string(opened) != plaintext {
			t.Errorf("round trip changed the value")
		}
	}
}

func TestEncryptNeverRepeatsItself(t *testing.T) {
	cipher := newCipher(t)

	first, err := cipher.Encrypt([]byte("same password"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	second, err := cipher.Encrypt([]byte("same password"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("the same plaintext must not produce the same ciphertext, otherwise equal passwords are visible in the database")
	}
}

func TestDecryptRejectsTamperedData(t *testing.T) {
	cipher := newCipher(t)

	sealed, err := cipher.Encrypt([]byte("registry password"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	cases := map[string][]byte{
		"flipped bit in the ciphertext": flip(sealed, len(sealed)-1),
		"flipped bit in the nonce":      flip(sealed, 0),
		"truncated":                     sealed[:len(sealed)-1],
		"shorter than a nonce":          sealed[:3],
		"empty":                         nil,
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := cipher.Decrypt(data); err == nil {
				t.Fatal("tampered ciphertext must be rejected, not silently decrypted")
			}
		})
	}
}

func TestDecryptRejectsAnotherInstancesKey(t *testing.T) {
	mine := newCipher(t)
	theirs := newCipher(t)

	sealed, err := mine.Encrypt([]byte("registry password"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if _, err := theirs.Decrypt(sealed); err == nil {
		t.Fatal("a database restored without its key file must not be readable")
	}
}

func newCipher(t *testing.T) *Cipher {
	t.Helper()
	cipher, err := LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatalf("LoadOrCreate: %v", err)
	}
	return cipher
}

func flip(data []byte, index int) []byte {
	out := bytes.Clone(data)
	out[index] ^= 0x01
	return out
}
