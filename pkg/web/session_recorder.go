package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SessionRecorder captures terminal I/O in asciinema format
type SessionRecorder struct {
	sessionID     string
	userID        string
	serverID      string
	file          *os.File
	startTime     time.Time
	bytesWritten  int64
	commandCount  int
	mutex         sync.Mutex
	logger        *zap.Logger
	recordingPath string
}

// AsciinemaHeader represents the asciinema v2 format header
type AsciinemaHeader struct {
	Version   int               `json:"version"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Timestamp int64             `json:"timestamp"`
	Env       map[string]string `json:"env,omitempty"`
}

// NewSessionRecorder creates a new session recorder
func NewSessionRecorder(sessionID, userID, serverID string, width, height int, logger *zap.Logger) (*SessionRecorder, error) {
	// Create recording directory
	recordDir := "/var/log/reignx/sessions"
	if err := os.MkdirAll(recordDir, 0755); err != nil {
		// Fallback to /tmp if /var/log is not writable
		recordDir = filepath.Join(os.TempDir(), "reignx", "sessions")
		if err := os.MkdirAll(recordDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create recording directory: %w", err)
		}
	}

	// Create recording file with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.cast", sessionID, timestamp)
	recordingPath := filepath.Join(recordDir, filename)

	file, err := os.Create(recordingPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create recording file: %w", err)
	}

	recorder := &SessionRecorder{
		sessionID:     sessionID,
		userID:        userID,
		serverID:      serverID,
		file:          file,
		startTime:     time.Now(),
		logger:        logger,
		recordingPath: recordingPath,
	}

	// Write asciinema v2 header
	header := AsciinemaHeader{
		Version:   2,
		Width:     width,
		Height:    height,
		Timestamp: time.Now().Unix(),
		Env: map[string]string{
			"TERM":  "xterm-256color",
			"SHELL": "/bin/bash",
		},
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to marshal header: %w", err)
	}

	if _, err := file.Write(headerJSON); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	if _, err := file.Write([]byte("\n")); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write newline: %w", err)
	}

	logger.Info("Session recorder created",
		zap.String("session_id", sessionID),
		zap.String("recording_path", recordingPath))

	return recorder, nil
}

// RecordOutput records output from the terminal
func (r *SessionRecorder) RecordOutput(data []byte) {
	if r == nil || r.file == nil {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	elapsed := time.Since(r.startTime).Seconds()

	// Asciinema v2 format: [time, "o", data]
	event := []interface{}{elapsed, "o", string(data)}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		r.logger.Error("Failed to marshal output event", zap.Error(err))
		return
	}

	if _, err := r.file.Write(eventJSON); err != nil {
		r.logger.Error("Failed to write output event", zap.Error(err))
		return
	}

	if _, err := r.file.Write([]byte("\n")); err != nil {
		r.logger.Error("Failed to write newline", zap.Error(err))
		return
	}

	r.bytesWritten += int64(len(data))
}

// RecordInput records input to the terminal
func (r *SessionRecorder) RecordInput(data []byte) {
	if r == nil || r.file == nil {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	elapsed := time.Since(r.startTime).Seconds()

	// Asciinema v2 format: [time, "i", data]
	event := []interface{}{elapsed, "i", string(data)}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		r.logger.Error("Failed to marshal input event", zap.Error(err))
		return
	}

	if _, err := r.file.Write(eventJSON); err != nil {
		r.logger.Error("Failed to write input event", zap.Error(err))
		return
	}

	if _, err := r.file.Write([]byte("\n")); err != nil {
		r.logger.Error("Failed to write newline", zap.Error(err))
		return
	}

	// Count commands (simple heuristic: newline in input)
	for _, b := range data {
		if b == '\n' || b == '\r' {
			r.commandCount++
		}
	}
}

// RecordResize records terminal resize events
func (r *SessionRecorder) RecordResize(rows, cols int) {
	if r == nil || r.file == nil {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.logger.Debug("Recording terminal resize",
		zap.Int("rows", rows),
		zap.Int("cols", cols))
}

// GetStats returns recording statistics
func (r *SessionRecorder) GetStats() (bytesWritten int64, commandCount int, duration time.Duration) {
	if r == nil {
		return 0, 0, 0
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.bytesWritten, r.commandCount, time.Since(r.startTime)
}

// GetRecordingPath returns the path to the recording file
func (r *SessionRecorder) GetRecordingPath() string {
	if r == nil {
		return ""
	}
	return r.recordingPath
}

// Close closes the recording file and returns statistics
func (r *SessionRecorder) Close() error {
	if r == nil || r.file == nil {
		return nil
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if err := r.file.Sync(); err != nil {
		r.logger.Warn("Failed to sync recording file", zap.Error(err))
	}

	if err := r.file.Close(); err != nil {
		return fmt.Errorf("failed to close recording file: %w", err)
	}

	duration := time.Since(r.startTime)

	r.logger.Info("Session recording closed",
		zap.String("session_id", r.sessionID),
		zap.String("recording_path", r.recordingPath),
		zap.Int64("bytes_written", r.bytesWritten),
		zap.Int("commands_executed", r.commandCount),
		zap.Duration("duration", duration))

	r.file = nil
	return nil
}
