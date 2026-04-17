package web

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/reignx/reignx/pkg/database/repository"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHExecutor handles remote command execution via SSH
type SSHExecutor struct {
	logger        *zap.Logger
	sshConfig     *SSHConfig
	serverRepo    repository.ServerRepository
	sshConfigRepo repository.SSHConfigRepository
}

// CommandResult holds the result of a command execution
type CommandResult struct {
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// NewSSHExecutor creates a new SSH executor
func NewSSHExecutor(logger *zap.Logger, sshConfig *SSHConfig, serverRepo repository.ServerRepository, sshConfigRepo repository.SSHConfigRepository) *SSHExecutor {
	return &SSHExecutor{
		logger:        logger,
		sshConfig:     sshConfig,
		serverRepo:    serverRepo,
		sshConfigRepo: sshConfigRepo,
	}
}

// ExecuteCommand executes a command on a remote server via SSH
func (e *SSHExecutor) ExecuteCommand(ctx context.Context, nodeID, command string, timeout time.Duration) (*CommandResult, error) {
	start := time.Now()

	e.logger.Info("Executing command",
		zap.String("node_id", nodeID),
		zap.String("command", command),
		zap.Duration("timeout", timeout))

	// Get server details
	server, err := e.serverRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	if server.IPAddress == "" {
		return nil, fmt.Errorf("server has no IP address")
	}

	// Create SSH connection
	client, err := e.createSSHClient(ctx, server.IPAddress, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Create session
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Execute command with timeout
	result := &CommandResult{
		ExitCode: 0,
	}

	done := make(chan error, 1)
	var output []byte

	go func() {
		output, err = session.CombinedOutput(command)
		done <- err
	}()

	// Wait for command or timeout
	select {
	case err := <-done:
		result.Duration = time.Since(start)

		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				result.ExitCode = exitErr.ExitStatus()
			} else {
				result.ExitCode = -1
				result.Error = err.Error()
			}
		}

		result.Stdout = string(output)

		e.logger.Info("Command execution completed",
			zap.String("node_id", nodeID),
			zap.Int("exit_code", result.ExitCode),
			zap.Duration("duration", result.Duration))

		return result, nil

	case <-time.After(timeout):
		// Try to kill the session
		session.Signal(ssh.SIGKILL)
		session.Close()

		result.Duration = time.Since(start)
		result.ExitCode = -1
		result.Error = fmt.Sprintf("command timeout after %v", timeout)

		e.logger.Warn("Command execution timeout",
			zap.String("node_id", nodeID),
			zap.Duration("timeout", timeout))

		return result, fmt.Errorf("command timeout after %v", timeout)

	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		session.Close()

		result.Duration = time.Since(start)
		result.ExitCode = -1
		result.Error = "context canceled"

		return result, ctx.Err()
	}
}

// createSSHClient creates an SSH client connection (reuses logic from terminal.go)
func (e *SSHExecutor) createSSHClient(ctx context.Context, serverIP string, nodeID string) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	// Try to get SSH config from database (with encrypted password)
	if e.sshConfigRepo != nil {
		// Try to get decrypted password from database
		password, err := e.sshConfigRepo.GetPassword(ctx, nodeID)
		if err == nil && password != "" {
			e.logger.Debug("Using encrypted password from database")
			authMethods = append(authMethods, ssh.Password(password))
		}
	}

	// Try SSH agent first (for macOS keychain keys)
	if agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		agentClient := agent.NewClient(agentConn)
		authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
	}

	// Try SSH key from file if provided
	if e.sshConfig.KeyPath != "" {
		key, err := os.ReadFile(e.sshConfig.KeyPath)
		if err == nil {
			if signer, err := ssh.ParsePrivateKey(key); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	// Try default SSH keys
	homeDir, err := os.UserHomeDir()
	if err == nil {
		for _, keyFile := range []string{"id_rsa", "id_ed25519", "id_ecdsa"} {
			keyPath := filepath.Join(homeDir, ".ssh", keyFile)
			if key, err := os.ReadFile(keyPath); err == nil {
				if signer, err := ssh.ParsePrivateKey(key); err == nil {
					authMethods = append(authMethods, ssh.PublicKeys(signer))
				}
			}
		}
	}

	// Add password auth if provided
	if e.sshConfig.Password != "" {
		authMethods = append(authMethods, ssh.Password(e.sshConfig.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH authentication methods available")
	}

	// Get host key callback with verification
	hostKeyCallback, err := e.sshConfigRepo.GetHostKeyCallback(ctx, nodeID)
	if err != nil {
		e.logger.Warn("Failed to get host key callback, using insecure mode",
			zap.String("node_id", nodeID),
			zap.Error(err))
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	sshClientConfig := &ssh.ClientConfig{
		User:            e.sshConfig.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         e.sshConfig.Timeout,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", serverIP), sshClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	return client, nil
}

// UploadFile uploads a file to the remote server via SCP
func (e *SSHExecutor) UploadFile(ctx context.Context, nodeID string, file io.Reader, targetPath string, fileSize int64) error {
	start := time.Now()

	e.logger.Info("Starting file upload",
		zap.String("node_id", nodeID),
		zap.String("target_path", targetPath),
		zap.Int64("size", fileSize))

	// Get server info
	server, err := e.serverRepo.GetByID(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// Create SSH client
	client, err := e.createSSHClient(ctx, server.IPAddress, nodeID)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Create SFTP session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Use SCP protocol to upload file
	// Create the target directory if it doesn't exist
	dirname := filepath.Dir(targetPath)
	if dirname != "" && dirname != "." {
		mkdirCmd := fmt.Sprintf("mkdir -p %s", dirname)
		if err := session.Run(mkdirCmd); err != nil {
			e.logger.Warn("Failed to create directory (may already exist)",
				zap.String("dir", dirname),
				zap.Error(err))
		}
		session.Close()

		// Create new session for SCP
		session, err = client.NewSession()
		if err != nil {
			return fmt.Errorf("failed to create SCP session: %w", err)
		}
		defer session.Close()
	}

	// Set up SCP command
	scpCmd := fmt.Sprintf("scp -t %s", targetPath)
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := session.Start(scpCmd); err != nil {
		return fmt.Errorf("failed to start SCP: %w", err)
	}

	// Wait for initial OK from server
	buf := make([]byte, 1)
	if _, err := stdout.Read(buf); err != nil {
		return fmt.Errorf("failed to read SCP response: %w", err)
	}
	if buf[0] != 0 {
		return fmt.Errorf("SCP server not ready")
	}

	// Send file header: C0644 <filesize> <filename>\n
	filename := filepath.Base(targetPath)
	header := fmt.Sprintf("C0644 %d %s\n", fileSize, filename)
	if _, err := stdin.Write([]byte(header)); err != nil {
		return fmt.Errorf("failed to write SCP header: %w", err)
	}

	// Wait for OK
	if _, err := stdout.Read(buf); err != nil {
		return fmt.Errorf("failed to read SCP response: %w", err)
	}
	if buf[0] != 0 {
		return fmt.Errorf("SCP rejected file header")
	}

	// Send file content
	written, err := io.Copy(stdin, file)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	// Send end of transfer marker
	if _, err := stdin.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to write SCP end marker: %w", err)
	}

	// Wait for final OK
	if _, err := stdout.Read(buf); err != nil {
		return fmt.Errorf("failed to read final SCP response: %w", err)
	}
	if buf[0] != 0 {
		return fmt.Errorf("SCP transfer failed")
	}

	stdin.Close()

	if err := session.Wait(); err != nil {
		return fmt.Errorf("SCP session failed: %w", err)
	}

	duration := time.Since(start)
	e.logger.Info("File uploaded successfully",
		zap.String("node_id", nodeID),
		zap.String("target_path", targetPath),
		zap.Int64("bytes_written", written),
		zap.Duration("duration", duration))

	return nil
}
