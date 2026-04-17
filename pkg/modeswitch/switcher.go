package modeswitch

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/reignx/reignx/pkg/core"
	"github.com/reignx/reignx/pkg/sshmode"
	"go.uber.org/zap"
)

// Config contains mode switcher configuration
type Config struct {
	AgentBinaryPath string        // Path to agent binary on control plane
	AgentInstallDir string        // Installation directory on target node
	AgentConfigTemplate string    // Agent configuration template
	SwitchTimeout   time.Duration // Timeout for switch operation
	Logger          *zap.Logger
}

// Switcher implements mode switching logic
type Switcher struct {
	config      *Config
	logger      *zap.Logger
	nodeRepo    core.NodeRepository
	taskRepo    core.TaskRepository
	sshPool     *sshmode.ConnectionPool
	progressMap sync.Map // map[nodeID]*core.SwitchProgress
}

// NewSwitcher creates a new mode switcher
func NewSwitcher(config *Config, nodeRepo core.NodeRepository, taskRepo core.TaskRepository, sshPool *sshmode.ConnectionPool) *Switcher {
	if config.SwitchTimeout == 0 {
		config.SwitchTimeout = 5 * time.Minute
	}

	return &Switcher{
		config:   config,
		logger:   config.Logger,
		nodeRepo: nodeRepo,
		taskRepo: taskRepo,
		sshPool:  sshPool,
	}
}

// SwitchToAgent upgrades a node from SSH mode to Agent mode
func (s *Switcher) SwitchToAgent(ctx context.Context, node *core.Node) error {
	s.logger.Info("Starting switch to agent mode",
		zap.String("node_id", node.ID),
		zap.String("hostname", node.Hostname),
	)

	// Check if switch is possible
	canSwitch, err := s.CanSwitch(ctx, node, core.NodeModeSSH, core.NodeModeAgent)
	if err != nil {
		return fmt.Errorf("switch validation failed: %w", err)
	}
	if !canSwitch {
		return fmt.Errorf("node cannot switch to agent mode")
	}

	// Initialize progress tracking
	progress := &core.SwitchProgress{
		NodeID:         node.ID,
		FromMode:       core.NodeModeSSH,
		ToMode:         core.NodeModeAgent,
		Status:         core.TaskStatusRunning,
		CurrentStep:    "Initializing",
		TotalSteps:     5,
		CompletedSteps: 0,
	}
	s.progressMap.Store(node.ID, progress)

	// Create a context with timeout
	switchCtx, cancel := context.WithTimeout(ctx, s.config.SwitchTimeout)
	defer cancel()

	// Execute switch steps
	steps := []struct {
		name string
		fn   func(context.Context, *core.Node) error
	}{
		{"Upload agent binary", s.uploadAgentBinary},
		{"Generate agent config", s.generateAgentConfig},
		{"Install agent service", s.installAgentService},
		{"Start agent", s.startAgent},
		{"Verify agent connection", s.verifyAgentConnection},
	}

	for i, step := range steps {
		progress.CurrentStep = step.name
		progress.CompletedSteps = i
		s.progressMap.Store(node.ID, progress)

		s.logger.Info("Executing switch step",
			zap.String("node_id", node.ID),
			zap.String("step", step.name),
			zap.Int("step_number", i+1),
			zap.Int("total_steps", len(steps)),
		)

		if err := step.fn(switchCtx, node); err != nil {
			progress.Status = core.TaskStatusFailed
			progress.Error = err.Error()
			s.progressMap.Store(node.ID, progress)

			return fmt.Errorf("switch failed at step '%s': %w", step.name, err)
		}
	}

	// Update node mode in database
	node.Mode = core.NodeModeAgent
	node.Status = core.NodeStatusOnline
	if err := s.nodeRepo.Update(ctx, node); err != nil {
		return fmt.Errorf("failed to update node mode: %w", err)
	}

	// Mark switch as completed
	progress.Status = core.TaskStatusCompleted
	progress.CompletedSteps = progress.TotalSteps
	progress.CurrentStep = "Completed"
	s.progressMap.Store(node.ID, progress)

	s.logger.Info("Successfully switched to agent mode",
		zap.String("node_id", node.ID),
		zap.String("hostname", node.Hostname),
	)

	return nil
}

// SwitchToSSH downgrades a node from Agent mode to SSH mode
func (s *Switcher) SwitchToSSH(ctx context.Context, node *core.Node) error {
	s.logger.Info("Starting switch to SSH mode",
		zap.String("node_id", node.ID),
		zap.String("hostname", node.Hostname),
	)

	// Check if switch is possible
	canSwitch, err := s.CanSwitch(ctx, node, core.NodeModeAgent, core.NodeModeSSH)
	if err != nil {
		return fmt.Errorf("switch validation failed: %w", err)
	}
	if !canSwitch {
		return fmt.Errorf("node cannot switch to SSH mode")
	}

	// Initialize progress tracking
	progress := &core.SwitchProgress{
		NodeID:         node.ID,
		FromMode:       core.NodeModeAgent,
		ToMode:         core.NodeModeSSH,
		Status:         core.TaskStatusRunning,
		CurrentStep:    "Initializing",
		TotalSteps:     3,
		CompletedSteps: 0,
	}
	s.progressMap.Store(node.ID, progress)

	// Create a context with timeout
	switchCtx, cancel := context.WithTimeout(ctx, s.config.SwitchTimeout)
	defer cancel()

	// Execute switch steps
	steps := []struct {
		name string
		fn   func(context.Context, *core.Node) error
	}{
		{"Stop agent service", s.stopAgent},
		{"Remove agent binary", s.removeAgentBinary},
		{"Verify SSH access", s.verifySSHAccess},
	}

	for i, step := range steps {
		progress.CurrentStep = step.name
		progress.CompletedSteps = i
		s.progressMap.Store(node.ID, progress)

		s.logger.Info("Executing switch step",
			zap.String("node_id", node.ID),
			zap.String("step", step.name),
			zap.Int("step_number", i+1),
			zap.Int("total_steps", len(steps)),
		)

		if err := step.fn(switchCtx, node); err != nil {
			progress.Status = core.TaskStatusFailed
			progress.Error = err.Error()
			s.progressMap.Store(node.ID, progress)

			return fmt.Errorf("switch failed at step '%s': %w", step.name, err)
		}
	}

	// Update node mode in database
	node.Mode = core.NodeModeSSH
	node.Status = core.NodeStatusOnline
	if err := s.nodeRepo.Update(ctx, node); err != nil {
		return fmt.Errorf("failed to update node mode: %w", err)
	}

	// Mark switch as completed
	progress.Status = core.TaskStatusCompleted
	progress.CompletedSteps = progress.TotalSteps
	progress.CurrentStep = "Completed"
	s.progressMap.Store(node.ID, progress)

	s.logger.Info("Successfully switched to SSH mode",
		zap.String("node_id", node.ID),
		zap.String("hostname", node.Hostname),
	)

	return nil
}

// CanSwitch checks if a node can switch modes
func (s *Switcher) CanSwitch(ctx context.Context, node *core.Node, fromMode, toMode core.NodeMode) (bool, error) {
	// Verify current mode matches expected from mode
	if node.Mode != fromMode {
		s.logger.Warn("Node mode mismatch",
			zap.String("node_id", node.ID),
			zap.String("expected", string(fromMode)),
			zap.String("actual", string(node.Mode)),
		)
		return false, fmt.Errorf("node is in %s mode, expected %s", node.Mode, fromMode)
	}

	// Check if node is online
	if node.Status != core.NodeStatusOnline {
		s.logger.Warn("Node is not online",
			zap.String("node_id", node.ID),
			zap.String("status", string(node.Status)),
		)
		return false, fmt.Errorf("node is not online (status: %s)", node.Status)
	}

	// Switching to Agent mode requires SSH access
	if toMode == core.NodeModeAgent {
		if node.SSHCredentials == nil {
			return false, fmt.Errorf("SSH credentials not configured for node")
		}

		// Test SSH connectivity
		if err := s.testSSHConnection(ctx, node); err != nil {
			return false, fmt.Errorf("SSH connection test failed: %w", err)
		}
	}

	// Check for pending tasks
	pendingTasks, err := s.taskRepo.GetPendingTasks(ctx, node.ID, 1)
	if err == nil && len(pendingTasks) > 0 {
		s.logger.Warn("Node has pending tasks",
			zap.String("node_id", node.ID),
			zap.Int("pending_count", len(pendingTasks)),
		)
		return false, fmt.Errorf("node has %d pending tasks", len(pendingTasks))
	}

	return true, nil
}

// GetProgress returns the progress of an ongoing mode switch
func (s *Switcher) GetProgress(ctx context.Context, nodeID string) (*core.SwitchProgress, error) {
	value, ok := s.progressMap.Load(nodeID)
	if !ok {
		return nil, fmt.Errorf("no switch operation in progress for node %s", nodeID)
	}

	progress, ok := value.(*core.SwitchProgress)
	if !ok {
		return nil, fmt.Errorf("invalid progress data")
	}

	return progress, nil
}

// Helper methods for switch steps

func (s *Switcher) uploadAgentBinary(ctx context.Context, node *core.Node) error {
	conn, err := s.sshPool.Get(node.IPAddress, node.SSHCredentials)
	if err != nil {
		return fmt.Errorf("failed to get SSH connection: %w", err)
	}
	defer s.sshPool.Release(conn)

	// Create installation directory
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	if err := session.Run(fmt.Sprintf("mkdir -p %s", s.config.AgentInstallDir)); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Upload agent binary via SFTP
	agentPath := filepath.Join(s.config.AgentInstallDir, "reignx-agent")
	if err := s.uploadFile(conn, s.config.AgentBinaryPath, agentPath); err != nil {
		return fmt.Errorf("failed to upload agent binary: %w", err)
	}

	// Make binary executable
	session2, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session2.Close()

	if err := session2.Run(fmt.Sprintf("chmod +x %s", agentPath)); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	return nil
}

func (s *Switcher) generateAgentConfig(ctx context.Context, node *core.Node) error {
	conn, err := s.sshPool.Get(node.IPAddress, node.SSHCredentials)
	if err != nil {
		return fmt.Errorf("failed to get SSH connection: %w", err)
	}
	defer s.sshPool.Release(conn)

	// Generate agent configuration
	config := fmt.Sprintf(`server_addr: "%s"
tls_enabled: false
heartbeat_interval: 30s
cache_dir: "/var/lib/reignx-agent"
max_concurrency: 10
log_level: "info"
`, "control-plane:50051") // TODO: Get actual control plane address from config

	configPath := filepath.Join(s.config.AgentInstallDir, "agent.yaml")

	// Upload config file
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	cmd := fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", configPath, config)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (s *Switcher) installAgentService(ctx context.Context, node *core.Node) error {
	conn, err := s.sshPool.Get(node.IPAddress, node.SSHCredentials)
	if err != nil {
		return fmt.Errorf("failed to get SSH connection: %w", err)
	}
	defer s.sshPool.Release(conn)

	// Detect init system and install service
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Create systemd service file
	serviceFile := `[Unit]
Description=ReignX Agent
After=network.target

[Service]
Type=simple
ExecStart=/opt/reignx-agent/reignx-agent -config /opt/reignx-agent/agent.yaml
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
`

	cmd := fmt.Sprintf("sudo bash -c 'cat > /etc/systemd/system/reignx-agent.service << \"EOF\"\n%sEOF'", serviceFile)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("failed to install service file: %w", err)
	}

	// Reload systemd
	session2, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session2.Close()

	if err := session2.Run("sudo systemctl daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable service
	session3, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session3.Close()

	if err := session3.Run("sudo systemctl enable reignx-agent"); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	return nil
}

func (s *Switcher) startAgent(ctx context.Context, node *core.Node) error {
	conn, err := s.sshPool.Get(node.IPAddress, node.SSHCredentials)
	if err != nil {
		return fmt.Errorf("failed to get SSH connection: %w", err)
	}
	defer s.sshPool.Release(conn)

	// Start the agent service
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	if err := session.Run("sudo systemctl start reignx-agent"); err != nil {
		return fmt.Errorf("failed to start agent service: %w", err)
	}

	// Wait a moment for agent to start
	time.Sleep(3 * time.Second)

	// Check service status
	session2, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session2.Close()

	if err := session2.Run("sudo systemctl is-active reignx-agent"); err != nil {
		return fmt.Errorf("agent service is not active: %w", err)
	}

	return nil
}

func (s *Switcher) verifyAgentConnection(ctx context.Context, node *core.Node) error {
	// Wait for agent to register with control plane
	maxWait := 60 * time.Second
	checkInterval := 5 * time.Second
	elapsed := time.Duration(0)

	for elapsed < maxWait {
		// Refresh node from database to check if agent has registered
		updatedNode, err := s.nodeRepo.Get(ctx, node.ID)
		if err != nil {
			return fmt.Errorf("failed to refresh node: %w", err)
		}

		// Check if node has sent heartbeat recently (within 1 minute)
		if time.Since(updatedNode.LastSeen) < time.Minute {
			s.logger.Info("Agent connection verified",
				zap.String("node_id", node.ID),
				zap.Time("last_seen", updatedNode.LastSeen),
			)
			return nil
		}

		time.Sleep(checkInterval)
		elapsed += checkInterval
	}

	return fmt.Errorf("agent failed to connect within %v", maxWait)
}

func (s *Switcher) stopAgent(ctx context.Context, node *core.Node) error {
	conn, err := s.sshPool.Get(node.IPAddress, node.SSHCredentials)
	if err != nil {
		return fmt.Errorf("failed to get SSH connection: %w", err)
	}
	defer s.sshPool.Release(conn)

	// Stop the agent service
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	if err := session.Run("sudo systemctl stop reignx-agent"); err != nil {
		// Log but don't fail if service is already stopped
		s.logger.Warn("Failed to stop agent service", zap.Error(err))
	}

	// Disable the service
	session2, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session2.Close()

	if err := session2.Run("sudo systemctl disable reignx-agent"); err != nil {
		s.logger.Warn("Failed to disable agent service", zap.Error(err))
	}

	return nil
}

func (s *Switcher) removeAgentBinary(ctx context.Context, node *core.Node) error {
	conn, err := s.sshPool.Get(node.IPAddress, node.SSHCredentials)
	if err != nil {
		return fmt.Errorf("failed to get SSH connection: %w", err)
	}
	defer s.sshPool.Release(conn)

	// Remove agent installation directory
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	if err := session.Run(fmt.Sprintf("sudo rm -rf %s", s.config.AgentInstallDir)); err != nil {
		return fmt.Errorf("failed to remove agent directory: %w", err)
	}

	// Remove systemd service file
	session2, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session2.Close()

	if err := session2.Run("sudo rm -f /etc/systemd/system/reignx-agent.service"); err != nil {
		s.logger.Warn("Failed to remove service file", zap.Error(err))
	}

	// Reload systemd
	session3, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session3.Close()

	if err := session3.Run("sudo systemctl daemon-reload"); err != nil {
		s.logger.Warn("Failed to reload systemd", zap.Error(err))
	}

	return nil
}

func (s *Switcher) verifySSHAccess(ctx context.Context, node *core.Node) error {
	return s.testSSHConnection(ctx, node)
}

func (s *Switcher) testSSHConnection(ctx context.Context, node *core.Node) error {
	conn, err := s.sshPool.Get(node.IPAddress, node.SSHCredentials)
	if err != nil {
		return fmt.Errorf("failed to connect via SSH: %w", err)
	}
	defer s.sshPool.Release(conn)

	// Test with a simple command
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	if err := session.Run("echo 'SSH connection test'"); err != nil {
		return fmt.Errorf("SSH command execution failed: %w", err)
	}

	return nil
}

func (s *Switcher) uploadFile(conn interface{}, srcPath, dstPath string) error {
	// TODO: Implement SFTP file upload
	// For now, this is a placeholder
	// Would use github.com/pkg/sftp for actual implementation
	return fmt.Errorf("file upload not yet implemented")
}
