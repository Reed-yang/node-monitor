package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	sshconfig "github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Pool manages persistent SSH connections to nodes.
type Pool struct {
	mu           sync.Mutex
	connections  map[string]*ssh.Client
	timeout      time.Duration
	user         string
	identityFile string
}

// NewPool creates a new SSH connection pool.
func NewPool(connectTimeout int, user, identityFile string) *Pool {
	return &Pool{
		connections:  make(map[string]*ssh.Client),
		timeout:      time.Duration(connectTimeout) * time.Second,
		user:         user,
		identityFile: identityFile,
	}
}

// getClient returns a cached or new SSH client for the given host.
func (p *Pool) getClient(host string) (*ssh.Client, error) {
	p.mu.Lock()
	if client, ok := p.connections[host]; ok {
		p.mu.Unlock()
		// Test if connection is still alive
		_, _, err := client.SendRequest("keepalive@node-monitor", true, nil)
		if err == nil {
			return client, nil
		}
		// Connection is dead, remove it
		p.mu.Lock()
		delete(p.connections, host)
		client.Close()
		p.mu.Unlock()
	} else {
		p.mu.Unlock()
	}

	// Resolve SSH config for this host
	user := p.user
	if user == "" {
		user = sshconfig.Get(host, "User")
	}
	if user == "" {
		user = os.Getenv("USER")
	}

	port := sshconfig.Get(host, "Port")
	if port == "" {
		port = "22"
	}

	hostname := sshconfig.Get(host, "Hostname")
	if hostname == "" {
		hostname = host
	}

	// Build auth methods
	var authMethods []ssh.AuthMethod

	// Try SSH agent first
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if agentConn, err := net.Dial("unix", sock); err == nil {
			authMethods = append(authMethods, ssh.PublicKeysCallback(agent.NewClient(agentConn).Signers))
		}
	}

	// Try identity file
	keyPath := p.identityFile
	if keyPath == "" {
		keyPath = sshconfig.Get(host, "IdentityFile")
	}
	if len(keyPath) > 0 && keyPath[0] == '~' {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, keyPath[1:])
	}
	keyPaths := []string{}
	if keyPath != "" {
		keyPaths = append(keyPaths, keyPath)
	}
	home, _ := os.UserHomeDir()
	keyPaths = append(keyPaths,
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
	)
	for _, kp := range keyPaths {
		if key, err := os.ReadFile(kp); err == nil {
			if signer, err := ssh.ParsePrivateKey(key); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				break
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH auth methods available (no agent, no key file)")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         p.timeout,
	}

	addr := net.JoinHostPort(hostname, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", host, err)
	}

	p.mu.Lock()
	p.connections[host] = client
	p.mu.Unlock()

	return client, nil
}

// RunCommand executes a command on the given host and returns stdout.
func (p *Pool) RunCommand(host, command string, cmdTimeout int) (string, error) {
	client, err := p.getClient(host)
	if err != nil {
		return "", err
	}

	session, err := client.NewSession()
	if err != nil {
		// Connection might be stale, discard and retry once
		p.mu.Lock()
		delete(p.connections, host)
		p.mu.Unlock()
		client.Close()

		client, err = p.getClient(host)
		if err != nil {
			return "", err
		}
		session, err = client.NewSession()
		if err != nil {
			return "", fmt.Errorf("SSH session %s: %w", host, err)
		}
	}
	defer session.Close()

	done := make(chan error, 1)
	var output []byte

	go func() {
		output, err = session.Output(command)
		done <- err
	}()

	select {
	case <-time.After(time.Duration(cmdTimeout) * time.Second):
		return "", fmt.Errorf("command timed out after %ds", cmdTimeout)
	case err := <-done:
		if err != nil {
			return string(output), fmt.Errorf("command failed on %s: %w", host, err)
		}
		return string(output), nil
	}
}

// Close closes all cached connections.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for host, client := range p.connections {
		client.Close()
		delete(p.connections, host)
	}
}
