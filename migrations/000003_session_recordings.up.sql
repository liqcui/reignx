-- Create session_recordings table for audit trail
CREATE TABLE IF NOT EXISTS session_recordings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id VARCHAR(255) NOT NULL UNIQUE,
    user_id VARCHAR(255) NOT NULL,
    server_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    duration INTEGER, -- seconds
    recording_path TEXT,
    playback_format VARCHAR(50) DEFAULT 'asciinema',
    bytes_transferred BIGINT DEFAULT 0,
    commands_executed INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_session_recordings_user ON session_recordings(user_id);
CREATE INDEX idx_session_recordings_server ON session_recordings(server_id);
CREATE INDEX idx_session_recordings_started ON session_recordings(started_at DESC);
CREATE INDEX idx_session_recordings_session ON session_recordings(session_id);

-- Comment on table
COMMENT ON TABLE session_recordings IS 'Stores metadata for terminal session recordings in asciinema format';
COMMENT ON COLUMN session_recordings.session_id IS 'Unique identifier for the WebSocket session';
COMMENT ON COLUMN session_recordings.recording_path IS 'File path to the asciinema recording file';
COMMENT ON COLUMN session_recordings.playback_format IS 'Format of the recording (asciinema v2)';
COMMENT ON COLUMN session_recordings.bytes_transferred IS 'Total bytes transferred during the session';
