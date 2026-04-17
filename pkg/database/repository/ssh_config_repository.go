package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/reignx/reignx/pkg/crypto"
	"github.com/reignx/reignx/pkg/observability/metrics"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHConfig represents SSH configuration for a node
type SSHConfig struct {
	ID                 string         `db:"id" json:"id"`
	NodeID             string         `db:"node_id" json:"node_id"`
	Host               string         `db:"host" json:"host"`
	Port               int            `db:"port" json:"port"`
	User               string         `db:"user" json:"user"`
	KeyPath            sql.NullString `db:"key_path" json:"key_path,omitempty"`
	PasswordEncrypted  sql.NullString `db:"password_encrypted" json:"-"` // Never expose encrypted password
	JumpHost           sql.NullString `db:"jump_host" json:"jump_host,omitempty"`
	Timeout            int            `db:"timeout" json:"timeout"`
	HostKeyFingerprint sql.NullString `db:"host_key_fingerprint" json:"host_key_fingerprint,omitempty"`
	HostKeyAlgorithm   sql.NullString `db:"host_key_algorithm" json:"host_key_algorithm,omitempty"`
	KnownHostsEntry    sql.NullString `db:"known_hosts_entry" json:"-"` // Do not expose in JSON
	CreatedAt          time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at" json:"updated_at"`
}

// SSHConfigRepository defines the interface for SSH config data access
type SSHConfigRepository interface {
	Create(ctx context.Context, config *SSHConfig) (*SSHConfig, error)
	GetByNodeID(ctx context.Context, nodeID string) (*SSHConfig, error)
	Update(ctx context.Context, config *SSHConfig) error
	Delete(ctx context.Context, id string) error
	GetPassword(ctx context.Context, nodeID string) (string, error)
	SetPassword(ctx context.Context, nodeID string, password string) error
	StoreHostKey(ctx context.Context, nodeID string, fingerprint, algorithm, knownHostsEntry string) error
	GetHostKeyCallback(ctx context.Context, nodeID string) (ssh.HostKeyCallback, error)
}

type sshConfigRepository struct {
	db        *sqlx.DB
	logger    *zap.Logger
	metrics   *metrics.Metrics
	encryptor *crypto.Encryptor
}

// NewSSHConfigRepository creates a new SSH config repository
func NewSSHConfigRepository(db *sqlx.DB, logger *zap.Logger, metrics *metrics.Metrics, encryptor *crypto.Encryptor) SSHConfigRepository {
	return &sshConfigRepository{
		db:        db,
		logger:    logger,
		metrics:   metrics,
		encryptor: encryptor,
	}
}

// Create creates a new SSH configuration
func (r *sshConfigRepository) Create(ctx context.Context, config *SSHConfig) (*SSHConfig, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "ssh_configs",
				"operation": "create",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "ssh_configs",
				"operation": "create",
			})
		}
	}()

	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	query := `
		INSERT INTO ssh_configs (id, node_id, host, port, "user", key_path, password_encrypted, jump_host, timeout, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, node_id, host, port, "user", key_path, jump_host, timeout, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		config.ID,
		config.NodeID,
		config.Host,
		config.Port,
		config.User,
		config.KeyPath,
		config.PasswordEncrypted,
		config.JumpHost,
		config.Timeout,
		config.CreatedAt,
		config.UpdatedAt,
	).Scan(
		&config.ID,
		&config.NodeID,
		&config.Host,
		&config.Port,
		&config.User,
		&config.KeyPath,
		&config.JumpHost,
		&config.Timeout,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create SSH config", zap.Error(err), zap.String("node_id", config.NodeID))
		return nil, fmt.Errorf("failed to create SSH config: %w", err)
	}

	r.logger.Info("SSH config created", zap.String("id", config.ID), zap.String("node_id", config.NodeID))
	return config, nil
}

// GetByNodeID retrieves SSH config by node ID
func (r *sshConfigRepository) GetByNodeID(ctx context.Context, nodeID string) (*SSHConfig, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "ssh_configs",
				"operation": "get_by_node_id",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "ssh_configs",
				"operation": "get_by_node_id",
			})
		}
	}()

	var config SSHConfig
	query := `
		SELECT id, node_id, host, port, "user", key_path, password_encrypted, jump_host, timeout,
		       host_key_fingerprint, host_key_algorithm, known_hosts_entry,
		       created_at, updated_at
		FROM ssh_configs
		WHERE node_id = $1
	`

	err := r.db.GetContext(ctx, &config, query, nodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("SSH config not found for node: %s", nodeID)
		}
		r.logger.Error("Failed to get SSH config", zap.Error(err), zap.String("node_id", nodeID))
		return nil, fmt.Errorf("failed to get SSH config: %w", err)
	}

	return &config, nil
}

// Update updates an SSH configuration
func (r *sshConfigRepository) Update(ctx context.Context, config *SSHConfig) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "ssh_configs",
				"operation": "update",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "ssh_configs",
				"operation": "update",
			})
		}
	}()

	config.UpdatedAt = time.Now()

	query := `
		UPDATE ssh_configs
		SET host = $2, port = $3, "user" = $4, key_path = $5, password_encrypted = $6,
		    jump_host = $7, timeout = $8, updated_at = $9
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		config.ID,
		config.Host,
		config.Port,
		config.User,
		config.KeyPath,
		config.PasswordEncrypted,
		config.JumpHost,
		config.Timeout,
		config.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to update SSH config", zap.Error(err), zap.String("id", config.ID))
		return fmt.Errorf("failed to update SSH config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("SSH config not found: %s", config.ID)
	}

	r.logger.Info("SSH config updated", zap.String("id", config.ID))
	return nil
}

// Delete deletes an SSH configuration
func (r *sshConfigRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "ssh_configs",
				"operation": "delete",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "ssh_configs",
				"operation": "delete",
			})
		}
	}()

	query := `DELETE FROM ssh_configs WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete SSH config", zap.Error(err), zap.String("id", id))
		return fmt.Errorf("failed to delete SSH config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("SSH config not found: %s", id)
	}

	r.logger.Info("SSH config deleted", zap.String("id", id))
	return nil
}

// GetPassword retrieves and decrypts the SSH password for a node
func (r *sshConfigRepository) GetPassword(ctx context.Context, nodeID string) (string, error) {
	config, err := r.GetByNodeID(ctx, nodeID)
	if err != nil {
		return "", err
	}

	if !config.PasswordEncrypted.Valid || config.PasswordEncrypted.String == "" {
		return "", nil
	}

	if r.encryptor == nil {
		r.logger.Warn("No encryptor configured, returning encrypted password as-is")
		return config.PasswordEncrypted.String, nil
	}

	password, err := r.encryptor.Decrypt(config.PasswordEncrypted.String)
	if err != nil {
		r.logger.Error("Failed to decrypt password", zap.Error(err), zap.String("node_id", nodeID))
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	return password, nil
}

// SetPassword encrypts and stores the SSH password for a node
func (r *sshConfigRepository) SetPassword(ctx context.Context, nodeID string, password string) error {
	if password == "" {
		return nil
	}

	if r.encryptor == nil {
		return fmt.Errorf("no encryptor configured")
	}

	encrypted, err := r.encryptor.Encrypt(password)
	if err != nil {
		r.logger.Error("Failed to encrypt password", zap.Error(err), zap.String("node_id", nodeID))
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	query := `
		UPDATE ssh_configs
		SET password_encrypted = $1, updated_at = $2
		WHERE node_id = $3
	`

	result, err := r.db.ExecContext(ctx, query, encrypted, time.Now(), nodeID)
	if err != nil {
		r.logger.Error("Failed to update password", zap.Error(err), zap.String("node_id", nodeID))
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("SSH config not found for node: %s", nodeID)
	}

	r.logger.Info("SSH password updated", zap.String("node_id", nodeID))
	return nil
}

// StoreHostKey stores the SSH host key fingerprint and details for a node
func (r *sshConfigRepository) StoreHostKey(ctx context.Context, nodeID string, fingerprint, algorithm, knownHostsEntry string) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "ssh_configs",
				"operation": "store_host_key",
			})
		}
	}()

	query := `
		UPDATE ssh_configs
		SET host_key_fingerprint = $1,
		    host_key_algorithm = $2,
		    known_hosts_entry = $3,
		    updated_at = $4
		WHERE node_id = $5
	`

	result, err := r.db.ExecContext(ctx, query, fingerprint, algorithm, knownHostsEntry, time.Now(), nodeID)
	if err != nil {
		r.logger.Error("Failed to store host key", zap.Error(err), zap.String("node_id", nodeID))
		return fmt.Errorf("failed to store host key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("SSH config not found for node: %s", nodeID)
	}

	r.logger.Info("SSH host key stored",
		zap.String("node_id", nodeID),
		zap.String("fingerprint", fingerprint),
		zap.String("algorithm", algorithm))

	return nil
}

// GetHostKeyCallback returns an SSH HostKeyCallback for the specified node
// Implements TOFU (Trust On First Use) pattern:
// - First connection: stores and trusts the host key
// - Subsequent connections: verifies against stored key
func (r *sshConfigRepository) GetHostKeyCallback(ctx context.Context, nodeID string) (ssh.HostKeyCallback, error) {
	config, err := r.GetByNodeID(ctx, nodeID)
	if err != nil {
		r.logger.Warn("SSH config not found, using insecure host key verification",
			zap.String("node_id", nodeID),
			zap.Error(err))
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// First connection - store and trust the host key
	if !config.HostKeyFingerprint.Valid || config.HostKeyFingerprint.String == "" {
		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			fingerprint := ssh.FingerprintSHA256(key)
			algorithm := key.Type()
			knownHostsEntry := knownhosts.Line([]string{hostname}, key)

			// Store the host key in database
			if err := r.StoreHostKey(ctx, nodeID, fingerprint, algorithm, knownHostsEntry); err != nil {
				r.logger.Error("Failed to store host key",
					zap.String("node_id", nodeID),
					zap.Error(err))
				return fmt.Errorf("failed to store host key: %w", err)
			}

			r.logger.Info("Stored new SSH host key (TOFU)",
				zap.String("node_id", nodeID),
				zap.String("hostname", hostname),
				zap.String("fingerprint", fingerprint),
				zap.String("algorithm", algorithm))

			return nil
		}, nil
	}

	// Subsequent connections - verify against stored key
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := ssh.FingerprintSHA256(key)
		algorithm := key.Type()

		if fingerprint != config.HostKeyFingerprint.String {
			r.logger.Error("SSH host key mismatch - possible MITM attack!",
				zap.String("node_id", nodeID),
				zap.String("hostname", hostname),
				zap.String("expected_fingerprint", config.HostKeyFingerprint.String),
				zap.String("actual_fingerprint", fingerprint),
				zap.String("expected_algorithm", config.HostKeyAlgorithm.String),
				zap.String("actual_algorithm", algorithm))

			return fmt.Errorf("host key verification failed: fingerprint mismatch (expected %s, got %s) - possible MITM attack",
				config.HostKeyFingerprint.String, fingerprint)
		}

		r.logger.Debug("SSH host key verified successfully",
			zap.String("node_id", nodeID),
			zap.String("fingerprint", fingerprint))

		return nil
	}, nil
}
