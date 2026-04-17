package pxe

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"
)

// HTTPServer serves kickstart files and OS images
type HTTPServer struct {
	root   string
	port   int
	server *http.Server
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(root string, port int) (*HTTPServer, error) {
	if port == 0 {
		port = 8080
	}

	return &HTTPServer{
		root: root,
		port: port,
	}, nil
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	// Serve kickstart files
	kickstartHandler := http.StripPrefix("/kickstart/",
		http.FileServer(http.Dir(filepath.Join(s.root, "kickstart"))))
	mux.Handle("/kickstart/", kickstartHandler)

	// Serve OS images
	imagesHandler := http.StripPrefix("/images/",
		http.FileServer(http.Dir(filepath.Join(s.root, "images"))))
	mux.Handle("/images/", imagesHandler)

	// Installation completion callback
	mux.HandleFunc("/complete/", s.handleComplete)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleComplete handles installation completion callbacks
func (s *HTTPServer) handleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		http.Error(w, "Missing server_id parameter", http.StatusBadRequest)
		return
	}

	// Create completion marker file
	markerPath := filepath.Join(s.root, "status", serverID+".done")
	if err := touchFile(markerPath); err != nil {
		http.Error(w, "Failed to create completion marker", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Installation completed"))
}

// touchFile creates an empty file or updates its timestamp
func touchFile(path string) error {
	file, err := http.Dir(filepath.Dir(path)).Open(filepath.Base(path))
	if err != nil {
		// File doesn't exist, create it
		f, err := http.Dir(filepath.Dir(path)).Open(filepath.Base(path))
		if err != nil {
			return err
		}
		defer f.Close()
	} else {
		defer file.Close()
	}
	return nil
}
