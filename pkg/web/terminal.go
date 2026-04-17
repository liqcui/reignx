package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/reignx/reignx/pkg/database/repository"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking in production
		return true
	},
}

// TerminalHandler handles WebSocket connections for server terminal sessions
type TerminalHandler struct {
	logger        *zap.Logger
	sshConfig     *SSHConfig
	serverRepo    repository.ServerRepository
	sshConfigRepo repository.SSHConfigRepository
	sessions      map[string]*TerminalSession
	sessionsMutex sync.RWMutex
}

// SSHConfig holds SSH connection configuration
type SSHConfig struct {
	Username string
	Password string
	KeyPath  string
	Timeout  time.Duration
}

// TerminalSession represents an active terminal session
type TerminalSession struct {
	ServerID   string
	SessionID  string
	SSHClient  *ssh.Client
	SSHSession *ssh.Session
	WSConn     *websocket.Conn
	StdinPipe  io.WriteCloser
	Recorder   *SessionRecorder
	Done       chan struct{}
}

// ResizeMessage represents a terminal resize message
type ResizeMessage struct {
	Type string `json:"type"`
	Rows int    `json:"rows"`
	Cols int    `json:"cols"`
}

// NewTerminalHandler creates a new terminal handler
func NewTerminalHandler(logger *zap.Logger, sshConfig *SSHConfig, serverRepo repository.ServerRepository, sshConfigRepo repository.SSHConfigRepository) *TerminalHandler {
	return &TerminalHandler{
		logger:        logger,
		sshConfig:     sshConfig,
		serverRepo:    serverRepo,
		sshConfigRepo: sshConfigRepo,
		sessions:      make(map[string]*TerminalSession),
	}
}

// HandleWebSocket handles WebSocket connections for terminal sessions
func (h *TerminalHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract server ID from URL path (e.g., /ws/terminal/{serverID})
	serverID := r.URL.Path[len("/ws/terminal/"):]
	if serverID == "" {
		http.Error(w, "Server ID required", http.StatusBadRequest)
		return
	}

	h.logger.Info("Terminal connection requested", zap.String("server_id", serverID))

	// Upgrade HTTP connection to WebSocket
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket", zap.Error(err))
		return
	}
	defer wsConn.Close()

	// Get server details from database
	// TODO: Implement actual database lookup
	serverIP, err := h.getServerIP(serverID)
	if err != nil {
		h.logger.Error("Failed to get server IP", zap.Error(err))
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v\r\n", err)))
		return
	}

	// Create SSH connection
	sshClient, err := h.createSSHConnection(serverIP, serverID)
	if err != nil {
		h.logger.Error("Failed to create SSH connection", zap.Error(err))
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("SSH connection failed: %v\r\n", err)))
		return
	}
	defer sshClient.Close()

	// Create SSH session
	sshSession, err := sshClient.NewSession()
	if err != nil {
		h.logger.Error("Failed to create SSH session", zap.Error(err))
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("SSH session failed: %v\r\n", err)))
		return
	}
	defer sshSession.Close()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// Request pseudo terminal
	if err := sshSession.RequestPty("xterm-256color", 40, 80, modes); err != nil {
		h.logger.Error("Failed to request PTY", zap.Error(err))
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("PTY request failed: %v\r\n", err)))
		return
	}

	// Get stdin pipe
	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		h.logger.Error("Failed to get stdin pipe", zap.Error(err))
		return
	}

	// Get stdout pipe
	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		h.logger.Error("Failed to get stdout pipe", zap.Error(err))
		return
	}

	// Get stderr pipe
	stderrPipe, err := sshSession.StderrPipe()
	if err != nil {
		h.logger.Error("Failed to get stderr pipe", zap.Error(err))
		return
	}

	// Start shell
	if err := sshSession.Shell(); err != nil {
		h.logger.Error("Failed to start shell", zap.Error(err))
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Shell start failed: %v\r\n", err)))
		return
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Create session recorder
	// TODO: Get userID from JWT token in request context
	userID := "unknown" // Placeholder until JWT middleware is integrated
	recorder, err := NewSessionRecorder(sessionID, userID, serverID, 80, 40, h.logger)
	if err != nil {
		h.logger.Warn("Failed to create session recorder", zap.Error(err))
		// Continue without recording rather than failing the session
	}

	// Create session
	session := &TerminalSession{
		ServerID:   serverID,
		SessionID:  sessionID,
		SSHClient:  sshClient,
		SSHSession: sshSession,
		WSConn:     wsConn,
		StdinPipe:  stdinPipe,
		Recorder:   recorder,
		Done:       make(chan struct{}),
	}

	// Store session
	h.sessionsMutex.Lock()
	h.sessions[serverID] = session
	h.sessionsMutex.Unlock()

	// Clean up session on exit
	defer func() {
		h.sessionsMutex.Lock()
		delete(h.sessions, serverID)
		h.sessionsMutex.Unlock()
		close(session.Done)
	}()

	// Start goroutines to handle I/O
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Read from SSH stdout and send to WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := stdoutPipe.Read(buf)
				if err != nil {
					if err != io.EOF {
						h.logger.Error("Error reading from SSH stdout", zap.Error(err))
					}
					cancel()
					return
				}
				if n > 0 {
					// Record output to session recording
					if session.Recorder != nil {
						session.Recorder.RecordOutput(buf[:n])
					}

					if err := wsConn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
						h.logger.Error("Error writing to WebSocket", zap.Error(err))
						cancel()
						return
					}
				}
			}
		}
	}()

	// Read from SSH stderr and send to WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := stderrPipe.Read(buf)
				if err != nil {
					if err != io.EOF {
						h.logger.Error("Error reading from SSH stderr", zap.Error(err))
					}
					return
				}
				if n > 0 {
					// Record stderr output to session recording
					if session.Recorder != nil {
						session.Recorder.RecordOutput(buf[:n])
					}

					if err := wsConn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
						h.logger.Error("Error writing to WebSocket", zap.Error(err))
						cancel()
						return
					}
				}
			}
		}
	}()

	// Read from WebSocket and send to SSH stdin
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msgType, data, err := wsConn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						h.logger.Error("WebSocket read error", zap.Error(err))
					}
					cancel()
					return
				}

				if msgType == websocket.TextMessage {
					// Check if it's a resize message
					var resizeMsg ResizeMessage
					if err := json.Unmarshal(data, &resizeMsg); err == nil && resizeMsg.Type == "resize" {
						// Resize terminal
						if err := sshSession.WindowChange(resizeMsg.Rows, resizeMsg.Cols); err != nil {
							h.logger.Error("Failed to resize terminal", zap.Error(err))
						}

						// Record resize event
						if session.Recorder != nil {
							session.Recorder.RecordResize(resizeMsg.Rows, resizeMsg.Cols)
						}
						continue
					}

					// Record input to session recording
					if session.Recorder != nil {
						session.Recorder.RecordInput(data)
					}

					// Regular input - send to SSH
					if _, err := stdinPipe.Write(data); err != nil {
						h.logger.Error("Error writing to SSH stdin", zap.Error(err))
						cancel()
						return
					}
				}
			}
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	// Close session recorder and log statistics
	if session.Recorder != nil {
		bytes, commands, duration := session.Recorder.GetStats()
		if err := session.Recorder.Close(); err != nil {
			h.logger.Error("Failed to close session recorder", zap.Error(err))
		}

		h.logger.Info("Terminal session closed",
			zap.String("server_id", serverID),
			zap.String("session_id", sessionID),
			zap.Int64("bytes_transferred", bytes),
			zap.Int("commands_executed", commands),
			zap.Duration("duration", duration))

		// TODO: Save session recording metadata to database
	} else {
		h.logger.Info("Terminal session closed", zap.String("server_id", serverID))
	}
}

// createSSHConnection creates an SSH connection to a server
func (h *TerminalHandler) createSSHConnection(serverIP string, nodeID string) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	// Try to get SSH config from database (with encrypted password)
	if h.sshConfigRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Try to get decrypted password from database
		password, err := h.sshConfigRepo.GetPassword(ctx, nodeID)
		if err == nil && password != "" {
			h.logger.Info("Using encrypted password from database")
			authMethods = append(authMethods, ssh.Password(password))
		}
	}

	// Try SSH agent first (for macOS keychain keys)
	if agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		agentClient := agent.NewClient(agentConn)
		authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
	}

	// Try SSH key from file if provided
	if h.sshConfig.KeyPath != "" {
		key, err := os.ReadFile(h.sshConfig.KeyPath)
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
	if h.sshConfig.Password != "" {
		authMethods = append(authMethods, ssh.Password(h.sshConfig.Password))
	}

	// Get host key callback with verification
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hostKeyCallback, err := h.sshConfigRepo.GetHostKeyCallback(ctx, nodeID)
	if err != nil {
		h.logger.Warn("Failed to get host key callback, using insecure mode",
			zap.String("node_id", nodeID),
			zap.Error(err))
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	config := &ssh.ClientConfig{
		User:            h.sshConfig.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         h.sshConfig.Timeout,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", serverIP), config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	return client, nil
}

// getServerIP retrieves the server IP address from the database
func (h *TerminalHandler) getServerIP(serverID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server, err := h.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return "", fmt.Errorf("server not found: %s", serverID)
	}

	if server.IPAddress == "" {
		return "", fmt.Errorf("server has no IP address: %s", serverID)
	}

	return server.IPAddress, nil
}

// CloseSession closes a terminal session
func (h *TerminalHandler) CloseSession(serverID string) error {
	h.sessionsMutex.Lock()
	defer h.sessionsMutex.Unlock()

	session, ok := h.sessions[serverID]
	if !ok {
		return fmt.Errorf("session not found: %s", serverID)
	}

	if session.WSConn != nil {
		session.WSConn.Close()
	}

	if session.SSHSession != nil {
		session.SSHSession.Close()
	}

	if session.SSHClient != nil {
		session.SSHClient.Close()
	}

	delete(h.sessions, serverID)

	return nil
}
