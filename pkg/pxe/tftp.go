package pxe

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pin/tftp"
)

// TFTPServer implements a TFTP server for PXE boot files
type TFTPServer struct {
	root   string
	port   int
	server *tftp.Server
}

// NewTFTPServer creates a new TFTP server
func NewTFTPServer(root string, port int) (*TFTPServer, error) {
	if port == 0 {
		port = 69 // Default TFTP port
	}

	return &TFTPServer{
		root: root,
		port: port,
	}, nil
}

// Start starts the TFTP server
func (s *TFTPServer) Start() error {
	server := tftp.NewServer(s.readHandler, s.writeHandler)
	server.SetTimeout(5 * time.Second)

	s.server = server

	go func() {
		addr := fmt.Sprintf(":%d", s.port)
		if err := server.ListenAndServe(addr); err != nil {
			fmt.Printf("TFTP server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the TFTP server
func (s *TFTPServer) Stop() error {
	if s.server != nil {
		s.server.Shutdown()
	}
	return nil
}

// readHandler handles TFTP read requests
func (s *TFTPServer) readHandler(filename string, rf io.ReaderFrom) error {
	// Sanitize filename
	filename = filepath.Clean(filename)
	if filepath.IsAbs(filename) {
		filename = filename[1:]
	}

	fullPath := filepath.Join(s.root, filename)

	// Check if file exists and is within root
	if !s.isPathSafe(fullPath) {
		return fmt.Errorf("access denied: %s", filename)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	n, err := rf.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("failed to send file: %w", err)
	}

	fmt.Printf("TFTP: Sent %s (%d bytes)\n", filename, n)
	return nil
}

// writeHandler handles TFTP write requests (disabled for security)
func (s *TFTPServer) writeHandler(filename string, wt io.WriterTo) error {
	return fmt.Errorf("write operations not supported")
}

// isPathSafe checks if a path is within the TFTP root directory
func (s *TFTPServer) isPathSafe(path string) bool {
	absRoot, err := filepath.Abs(s.root)
	if err != nil {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}

	// Path must not escape root directory
	return !filepath.IsAbs(rel) && rel != ".." && len(rel) >= 0
}
