package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/imjasonh/infinite-git/internal/pktline"
	"github.com/imjasonh/infinite-git/internal/protocol"
)

// handleInfoRefs handles the reference discovery phase.
func (s *Server) handleInfoRefs(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")

	// Only support git-upload-pack (fetch/clone)
	if service != "git-upload-pack" {
		http.Error(w, "Service not supported", http.StatusForbidden)
		return
	}

	// Generate a new commit before advertising refs
	s.mu.Lock()
	commitSHA, err := s.generator.GenerateCommit()
	s.mu.Unlock()

	if err != nil {
		s.logger.Error("failed to generate commit", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.logger.Info("generated new commit", "sha", commitSHA, "counter", s.generator.GetCounter())

	// Set headers
	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
	w.Header().Set("Cache-Control", "no-cache")

	// Write response
	pw := pktline.NewWriter(w)

	// Service declaration
	if err := pw.Writef("# service=%s\n", service); err != nil {
		s.logger.Error("failed to write service line", "error", err)
		return
	}
	if err := pw.Flush(); err != nil {
		s.logger.Error("failed to write flush", "error", err)
		return
	}

	// Get current refs
	refs, err := s.repo.GetRefs()
	if err != nil {
		s.logger.Error("failed to get refs", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Write capabilities with first ref
	capabilities := strings.Join(s.repo.GetCapabilities(), " ")
	first := true

	for ref, oid := range refs {
		if first {
			if err := pw.Writef("%s %s\x00%s\n", oid, ref, capabilities); err != nil {
				s.logger.Error("failed to write ref with capabilities", "error", err)
				return
			}
			first = false
		} else {
			if err := pw.Writef("%s %s\n", oid, ref); err != nil {
				s.logger.Error("failed to write ref", "error", err)
				return
			}
		}
	}

	// Final flush
	if err := pw.Flush(); err != nil {
		s.logger.Error("failed to write final flush", "error", err)
		return
	}
}

// handleUploadPack handles the pack upload phase.
func (s *Server) handleUploadPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
	w.Header().Set("Cache-Control", "no-cache")

	// Create upload-pack handler
	up := protocol.NewUploadPack(s.repo)

	// Process the request
	if err := up.HandleRequest(r.Body, w); err != nil {
		s.logger.Error("upload-pack failed", "error", err)
		// Don't send HTTP error here as we may have already started writing response
		return
	}

	s.logger.Info("completed upload-pack")
}
