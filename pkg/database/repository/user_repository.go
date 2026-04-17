package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/reignx/reignx/pkg/database/models"
	"github.com/reignx/reignx/pkg/observability/metrics"
	"go.uber.org/zap"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *models.User) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*models.User, error)
	UpdateLastLogin(ctx context.Context, userID string) error
}

type userRepository struct {
	db      *sqlx.DB
	logger  *zap.Logger
	metrics *metrics.Metrics
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB, logger *zap.Logger, metrics *metrics.Metrics) UserRepository {
	return &userRepository{
		db:      db,
		logger:  logger,
		metrics: metrics,
	}
}

// Create creates a new user
func (r *userRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "users",
				"operation": "create",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "users",
				"operation": "create",
			})
		}
	}()

	// Generate UUID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
		INSERT INTO users (id, username, email, password_hash, role, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, username, email, role, enabled, last_login, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.Enabled,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.Enabled,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create user", zap.Error(err), zap.String("username", user.Username))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.Info("User created", zap.String("user_id", user.ID), zap.String("username", user.Username))
	return user, nil
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "users",
				"operation": "get_by_id",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "users",
				"operation": "get_by_id",
			})
		}
	}()

	var user models.User
	query := `
		SELECT id, username, email, password_hash, role, enabled, last_login, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		r.logger.Error("Failed to get user by ID", zap.Error(err), zap.String("user_id", id))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "users",
				"operation": "get_by_username",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "users",
				"operation": "get_by_username",
			})
		}
	}()

	var user models.User
	query := `
		SELECT id, username, email, password_hash, role, enabled, last_login, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		r.logger.Error("Failed to get user by username", zap.Error(err), zap.String("username", username))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "users",
				"operation": "get_by_email",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "users",
				"operation": "get_by_email",
			})
		}
	}()

	var user models.User
	query := `
		SELECT id, username, email, password_hash, role, enabled, last_login, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", email)
		}
		r.logger.Error("Failed to get user by email", zap.Error(err), zap.String("email", email))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// Update updates a user
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "users",
				"operation": "update",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "users",
				"operation": "update",
			})
		}
	}()

	user.UpdatedAt = time.Now()

	query := `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, role = $5, enabled = $6, updated_at = $7
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.Enabled,
		user.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to update user", zap.Error(err), zap.String("user_id", user.ID))
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", user.ID)
	}

	r.logger.Info("User updated", zap.String("user_id", user.ID))
	return nil
}

// Delete deletes a user
func (r *userRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "users",
				"operation": "delete",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "users",
				"operation": "delete",
			})
		}
	}()

	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete user", zap.Error(err), zap.String("user_id", id))
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	r.logger.Info("User deleted", zap.String("user_id", id))
	return nil
}

// List retrieves all users
func (r *userRepository) List(ctx context.Context) ([]*models.User, error) {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "users",
				"operation": "list",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "users",
				"operation": "list",
			})
		}
	}()

	var users []*models.User
	query := `
		SELECT id, username, email, password_hash, role, enabled, last_login, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &users, query)
	if err != nil {
		r.logger.Error("Failed to list users", zap.Error(err))
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *userRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	start := time.Now()
	defer func() {
		if r.metrics != nil {
			duration := time.Since(start).Seconds()
			r.metrics.RecordHistogram("db_query_duration_seconds", duration, map[string]string{
				"table":     "users",
				"operation": "update_last_login",
			})
			r.metrics.RecordCounter("db_queries_total", 1, map[string]string{
				"table":     "users",
				"operation": "update_last_login",
			})
		}
	}()

	query := `
		UPDATE users
		SET last_login = $2, updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, userID, now)
	if err != nil {
		r.logger.Error("Failed to update last login", zap.Error(err), zap.String("user_id", userID))
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}

	return nil
}
