package companion

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPrepareConnectionNeverReplacesExistingKey(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("OpenSSH unavailable")
	}
	directory := filepath.Join(t.TempDir(), "connection")
	public, err := PrepareVMConnection(directory)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(directory, "id_ed25519"))
	if err != nil {
		t.Fatal(err)
	}
	pub, err := os.ReadFile(public)
	if err != nil || !bytes.HasPrefix(pub, []byte("ssh-ed25519 ")) {
		t.Fatal("invalid public key")
	}
	if _, err = PrepareVMConnection(directory); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(directory, "id_ed25519"))
	if !bytes.Equal(before, after) {
		t.Fatal("private key silently rotated")
	}
	if err = os.Remove(public); err != nil {
		t.Fatal(err)
	}
	if _, err = PrepareVMConnection(directory); err == nil {
		t.Fatal("missing public key should require attention")
	}
	after, _ = os.ReadFile(filepath.Join(directory, "id_ed25519"))
	if !bytes.Equal(before, after) {
		t.Fatal("existing identity replaced")
	}
}
func TestVMTunnelRequiresExplicitPreparation(t *testing.T) {
	if _, stop, err := StartVMTunnel(t.TempDir(), nil); err == nil || stop != nil {
		t.Fatal("tunnel started without identity")
	}
}
