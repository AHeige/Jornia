package companion

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Public VM host key read through the existing trusted development SSH connection.
// This private-test launcher never accepts a changed server key automatically.
const vmHostKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIvdac3P8T7U9eaWvYCcAngbgku5v0Ma7w72xVbt9ZqR"

func PrepareVMConnection(directory string) (string, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		account, err := user.Current()
		if err != nil {
			return "", err
		}
		cmd := exec.Command("icacls.exe", directory, "/inheritance:r", "/grant:r", "*"+account.Uid+":(OI)(CI)F", "*S-1-5-18:(OI)(CI)F")
		if err = cmd.Run(); err != nil {
			return "", errors.New("could not protect the Jornia connection directory")
		}
	}
	key := filepath.Join(directory, "id_ed25519")
	if _, err := os.Stat(key); os.IsNotExist(err) {
		cmd := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "wisp-windows-api-only", "-f", key)
		if err = cmd.Run(); err != nil {
			return "", errors.New("SSH key generation failed; Windows OpenSSH Client must be available")
		}
	} else if err != nil {
		return "", err
	}
	if _, err := os.Stat(key + ".pub"); err != nil {
		return "", errors.New("public key missing; existing private key left unchanged")
	}
	return key + ".pub", nil
}

// StartVMTunnel forwards a random local loopback port to only the Jornia API.
// It uses the Windows host's existing VirtualBox SSH mapping, not a new public port.
func StartVMTunnel(directory string, stderr io.Writer) (string, func(), error) {
	key := filepath.Join(directory, "id_ed25519")
	if _, err := os.Stat(key); err != nil {
		return "", nil, errors.New("prepare the Jornia connection with wisp.exe -prepare-vm and register its public key first")
	}
	known := filepath.Join(directory, "known_hosts")
	if err := os.WriteFile(known, []byte("[127.0.0.1]:2222 "+vmHostKey+"\n"), 0600); err != nil {
		return "", nil, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	address := listener.Addr().String()
	listener.Close()
	port := strings.TrimPrefix(address, "127.0.0.1:")
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "ssh", "-N", "-T", "-p", "2222", "-i", key, "-o", "UserKnownHostsFile="+known, "-o", "StrictHostKeyChecking=yes", "-o", "IdentitiesOnly=yes", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-o", "ExitOnForwardFailure=yes", "-o", "ServerAliveInterval=20", "-o", "ServerAliveCountMax=2", "-L", "127.0.0.1:"+port+":127.0.0.1:18090", "leon@127.0.0.1")
	cmd.Stderr = stderr
	if err = cmd.Start(); err != nil {
		cancel()
		return "", nil, errors.New("could not start OpenSSH Client")
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	stop := func() { cancel() }
	backend := "http://" + address
	client := http.Client{Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for i := 0; i < 30; i++ {
		select {
		case <-done:
			cancel()
			return "", nil, errors.New("Jornia SSH connection failed; check public key registration and VM status")
		default:
		}
		response, err := client.Get(backend + "/healthz")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == 200 {
				return backend, stop, nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	cancel()
	return "", nil, fmt.Errorf("Jornia API did not become reachable; no model request was sent")
}
