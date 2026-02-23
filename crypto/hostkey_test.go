package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestLoadOrGenerateHostKey_GeneratesNew(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_host_key")

	signer, err := LoadOrGenerateHostKey(path)
	if err != nil {
		t.Fatalf("LoadOrGenerateHostKey() error: %v", err)
	}
	if signer == nil {
		t.Fatal("LoadOrGenerateHostKey() returned nil signer")
	}
	if signer.PublicKey().Type() != "ssh-ed25519" {
		t.Errorf("expected ssh-ed25519 key, got %s", signer.PublicKey().Type())
	}

	// Verify file was written
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("generated key file not found: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("key file permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestLoadOrGenerateHostKey_LoadsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_host_key")

	// Generate first
	signer1, err := LoadOrGenerateHostKey(path)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Load existing
	signer2, err := LoadOrGenerateHostKey(path)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	// Should be the same key
	pub1 := ssh.MarshalAuthorizedKey(signer1.PublicKey())
	pub2 := ssh.MarshalAuthorizedKey(signer2.PublicKey())
	if string(pub1) != string(pub2) {
		t.Error("loaded key does not match generated key")
	}
}

func TestLoadOrGenerateHostKey_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_host_key")

	// Write garbage
	os.WriteFile(path, []byte("not a valid key"), 0600)

	// Should generate a new key (overwriting the corrupt file)
	signer, err := LoadOrGenerateHostKey(path)
	if err != nil {
		t.Fatalf("LoadOrGenerateHostKey() with corrupt file error: %v", err)
	}
	if signer == nil {
		t.Fatal("returned nil signer for corrupt file")
	}
}

func TestLoadOrGenerateHostKey_UnwritablePath(t *testing.T) {
	// Non-existent deeply nested path that can't be written
	path := "/nonexistent/deeply/nested/path/host_key"
	_, err := LoadOrGenerateHostKey(path)
	if err == nil {
		t.Fatal("expected error for unwritable path, got nil")
	}
}
