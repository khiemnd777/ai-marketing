package cryptox

import (
	"bytes"
	"testing"
)

func TestCipherBindsWorkspaceAssociatedData(t *testing.T) {
	cipher, err := New(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, nonce, err := cipher.Encrypt([]byte("meta-access-token"), "workspace:one")
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := cipher.Decrypt(ciphertext, nonce, "workspace:one")
	if err != nil || string(plaintext) != "meta-access-token" {
		t.Fatalf("round trip failed: value=%q error=%v", plaintext, err)
	}
	if _, err = cipher.Decrypt(ciphertext, nonce, "workspace:two"); err == nil {
		t.Fatal("ciphertext must not decrypt under another workspace")
	}
}
