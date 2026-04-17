package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reignx/reignx/pkg/core"
)

// NodeRepository implements core.NodeRepository
type NodeRepository struct {
	db *DB
}

// NewNodeRepository creates a new node repository
func NewNodeRepository(db *DB) *NodeRepository {
	return &NodeRepository{db: db}
}

// Create adds a new node
func (r *NodeRepository) Create(ctx context.Context, node *core.Node) error {
	tagsJSON, err := json.Marshal(node.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	metadataJSON, err := json.Marshal(node.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Handle optional MAC address
	var macAddress sql.NullString
	if node.MACAddress != "" {
		macAddress = sql.NullString{String: node.MACAddress, Valid: true}
	}

	query := `
		INSERT INTO nodes (
			hostname, ip_address, mac_address, mode, status,
			os_type, os_version, architecture, agent_version,
			last_seen, tags, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id`

	err = r.db.QueryRowContext(ctx, query,
		node.Hostname, node.IPAddress, macAddress,
		node.Mode, node.Status, node.OSType, node.OSVersion,
		node.Architecture, node.AgentVersion, node.LastSeen,
		tagsJSON, metadataJSON,
	).Scan(&node.ID)

	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	// Create SSH config if provided
	if node.SSHConfig != nil {
		if err := r.createSSHConfig(ctx, node.ID, node.SSHConfig); err != nil {
			return err
		}
	}

	// Create BMC config if provided
	if node.BMCConfig != nil {
		if err := r.createBMCConfig(ctx, node.ID, node.BMCConfig); err != nil {
			return err
		}
	}

	return nil
}

// Get retrieves a node by ID
func (r *NodeRepository) Get(ctx context.Context, id string) (*core.Node, error) {
	query := `
		SELECT
			id, hostname, ip_address, mac_address, mode, status,
			os_type, os_version, architecture, agent_version,
			last_seen, tags, metadata, created_at, updated_at
		FROM nodes
		WHERE id = $1`

	var node core.Node
	var tagsJSON, metadataJSON []byte
	var macAddress sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&node.ID, &node.Hostname, &node.IPAddress, &macAddress,
		&node.Mode, &node.Status, &node.OSType, &node.OSVersion,
		&node.Architecture, &node.AgentVersion, &node.LastSeen,
		&tagsJSON, &metadataJSON, &node.CreatedAt, &node.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("node not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if macAddress.Valid {
		node.MACAddress = macAddress.String
	}

	if err := json.Unmarshal(tagsJSON, &node.Tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &node.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Load SSH config
	node.SSHConfig, _ = r.getSSHConfig(ctx, id)

	// Load BMC config
	node.BMCConfig, _ = r.getBMCConfig(ctx, id)

	return &node, nil
}

// GetByIP retrieves a node by IP address
func (r *NodeRepository) GetByIP(ctx context.Context, ip string) (*core.Node, error) {
	query := `SELECT id FROM nodes WHERE ip_address = $1`

	var id string
	err := r.db.QueryRowContext(ctx, query, ip).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("node not found with IP: %s", ip)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get node by IP: %w", err)
	}

	return r.Get(ctx, id)
}

// GetByHostname retrieves a node by hostname
func (r *NodeRepository) GetByHostname(ctx context.Context, hostname string) (*core.Node, error) {
	query := `SELECT id FROM nodes WHERE hostname = $1`

	var id string
	err := r.db.QueryRowContext(ctx, query, hostname).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("node not found with hostname: %s", hostname)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get node by hostname: %w", err)
	}

	return r.Get(ctx, id)
}

// List retrieves nodes matching the filter
func (r *NodeRepository) List(ctx context.Context, filter *core.NodeFilter) ([]*core.Node, error) {
	query := `
		SELECT
			id, hostname, ip_address, mac_address, mode, status,
			os_type, os_version, architecture, agent_version,
			last_seen, tags, metadata, created_at, updated_at
		FROM nodes
		WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if len(filter.IDs) > 0 {
		query += fmt.Sprintf(" AND id = ANY($%d)", argCount)
		args = append(args, filter.IDs)
		argCount++
	}

	if len(filter.Hostnames) > 0 {
		query += fmt.Sprintf(" AND hostname = ANY($%d)", argCount)
		args = append(args, filter.Hostnames)
		argCount++
	}

	if filter.Mode != "" {
		query += fmt.Sprintf(" AND mode = $%d", argCount)
		args = append(args, filter.Mode)
		argCount++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	if filter.OSType != "" {
		query += fmt.Sprintf(" AND os_type = $%d", argCount)
		args = append(args, filter.OSType)
		argCount++
	}

	// Order and pagination
	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
		argCount++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*core.Node
	for rows.Next() {
		var node core.Node
		var tagsJSON, metadataJSON []byte
		var macAddress sql.NullString

		err := rows.Scan(
			&node.ID, &node.Hostname, &node.IPAddress, &macAddress,
			&node.Mode, &node.Status, &node.OSType, &node.OSVersion,
			&node.Architecture, &node.AgentVersion, &node.LastSeen,
			&tagsJSON, &metadataJSON, &node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		if macAddress.Valid {
			node.MACAddress = macAddress.String
		}

		if err := json.Unmarshal(tagsJSON, &node.Tags); err != nil {
			continue
		}

		if err := json.Unmarshal(metadataJSON, &node.Metadata); err != nil {
			continue
		}

		nodes = append(nodes, &node)
	}

	return nodes, nil
}

// Update updates an existing node
func (r *NodeRepository) Update(ctx context.Context, node *core.Node) error {
	tagsJSON, err := json.Marshal(node.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	metadataJSON, err := json.Marshal(node.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Handle optional MAC address
	var macAddress sql.NullString
	if node.MACAddress != "" {
		macAddress = sql.NullString{String: node.MACAddress, Valid: true}
	}

	query := `
		UPDATE nodes SET
			hostname = $2,
			ip_address = $3,
			mac_address = $4,
			mode = $5,
			status = $6,
			os_type = $7,
			os_version = $8,
			architecture = $9,
			agent_version = $10,
			last_seen = $11,
			tags = $12,
			metadata = $13
		WHERE id = $1`

	_, err = r.db.ExecContext(ctx, query,
		node.ID, node.Hostname, node.IPAddress, macAddress,
		node.Mode, node.Status, node.OSType, node.OSVersion,
		node.Architecture, node.AgentVersion, node.LastSeen,
		tagsJSON, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	return nil
}

// UpdateStatus updates only the status and last_seen
func (r *NodeRepository) UpdateStatus(ctx context.Context, id string, status core.NodeStatus, lastSeen time.Time) error {
	query := `UPDATE nodes SET status = $2, last_seen = $3 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, status, lastSeen)
	if err != nil {
		return fmt.Errorf("failed to update node status: %w", err)
	}

	return nil
}

// UpdateMode changes the node's management mode
func (r *NodeRepository) UpdateMode(ctx context.Context, id string, mode core.NodeMode) error {
	query := `UPDATE nodes SET mode = $2 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, mode)
	if err != nil {
		return fmt.Errorf("failed to update node mode: %w", err)
	}

	return nil
}

// Delete removes a node
func (r *NodeRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM nodes WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

// Count returns the total number of nodes matching the filter
func (r *NodeRepository) Count(ctx context.Context, filter *core.NodeFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM nodes WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	if filter.Mode != "" {
		query += fmt.Sprintf(" AND mode = $%d", argCount)
		args = append(args, filter.Mode)
		argCount++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count nodes: %w", err)
	}

	return count, nil
}

// AddTags adds or updates tags for a node
func (r *NodeRepository) AddTags(ctx context.Context, id string, tags map[string]string) error {
	node, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	if node.Tags == nil {
		node.Tags = make(map[string]string)
	}

	for k, v := range tags {
		node.Tags[k] = v
	}

	return r.Update(ctx, node)
}

// RemoveTags removes tags from a node
func (r *NodeRepository) RemoveTags(ctx context.Context, id string, tagKeys []string) error {
	node, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	if node.Tags == nil {
		return nil
	}

	for _, key := range tagKeys {
		delete(node.Tags, key)
	}

	return r.Update(ctx, node)
}

// Helper functions

func (r *NodeRepository) createSSHConfig(ctx context.Context, nodeID string, config *core.SSHConfig) error {
	query := `
		INSERT INTO ssh_configs (
			node_id, host, port, user, key_path, password_encrypted, jump_host, timeout
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		nodeID, config.Host, config.Port, config.User,
		config.KeyPath, config.Password, config.JumpHost, config.Timeout,
	)

	if err != nil {
		return fmt.Errorf("failed to create SSH config: %w", err)
	}

	return nil
}

func (r *NodeRepository) getSSHConfig(ctx context.Context, nodeID string) (*core.SSHConfig, error) {
	query := `
		SELECT host, port, user, key_path, password_encrypted, jump_host, timeout
		FROM ssh_configs
		WHERE node_id = $1`

	var config core.SSHConfig
	var keyPath, password, jumpHost sql.NullString

	err := r.db.QueryRowContext(ctx, query, nodeID).Scan(
		&config.Host, &config.Port, &config.User,
		&keyPath, &password, &jumpHost, &config.Timeout,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH config: %w", err)
	}

	if keyPath.Valid {
		config.KeyPath = keyPath.String
	}
	if password.Valid {
		config.Password = password.String
	}
	if jumpHost.Valid {
		config.JumpHost = jumpHost.String
	}

	return &config, nil
}

func (r *NodeRepository) createBMCConfig(ctx context.Context, nodeID string, config *core.BMCConfig) error {
	query := `
		INSERT INTO bmc_configs (node_id, address, user, password_encrypted, type)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.ExecContext(ctx, query,
		nodeID, config.Address, config.User, config.Password, config.Type,
	)

	if err != nil {
		return fmt.Errorf("failed to create BMC config: %w", err)
	}

	return nil
}

func (r *NodeRepository) getBMCConfig(ctx context.Context, nodeID string) (*core.BMCConfig, error) {
	query := `
		SELECT address, user, password_encrypted, type
		FROM bmc_configs
		WHERE node_id = $1`

	var config core.BMCConfig

	err := r.db.QueryRowContext(ctx, query, nodeID).Scan(
		&config.Address, &config.User, &config.Password, &config.Type,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get BMC config: %w", err)
	}

	return &config, nil
}
