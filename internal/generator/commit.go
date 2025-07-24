package generator

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/imjasonh/infinite-git/internal/object"
	"github.com/imjasonh/infinite-git/internal/repo"
)

// Generator creates new commits on demand.
type Generator struct {
	repo    *repo.Repository
	counter int64
}

// New creates a new commit generator.
func New(r *repo.Repository) *Generator {
	return &Generator{
		repo: r,
	}
}

// GenerateCommit creates a new commit and updates the main branch.
func (g *Generator) GenerateCommit() (string, error) {
	// Increment counter atomically
	count := atomic.AddInt64(&g.counter, 1)

	// Get current HEAD commit
	refs, err := g.repo.GetRefs()
	if err != nil {
		return "", fmt.Errorf("getting refs: %w", err)
	}

	parentHash := refs["refs/heads/main"]
	if parentHash == "" {
		return "", fmt.Errorf("main branch not found")
	}

	// Read parent commit to get its tree
	parentData, err := g.repo.ReadObject(parentHash)
	if err != nil {
		return "", fmt.Errorf("reading parent commit: %w", err)
	}

	// Parse parent commit to find tree hash
	var parentTreeHash string
	lines := splitLines(string(parentData))
	for _, line := range lines {
		if len(line) > 5 && line[:5] == "tree " {
			parentTreeHash = line[5:]
			break
		}
	}

	// Read parent tree
	parentTreeData, err := g.repo.ReadObject(parentTreeHash)
	if err != nil {
		return "", fmt.Errorf("reading parent tree: %w", err)
	}

	// Parse existing tree entries
	existingEntries := parseTree(parentTreeData)

	// Create new file content
	filename := fmt.Sprintf("pull_%d.txt", count)
	content := fmt.Sprintf("Pull request #%d\nTimestamp: %s\n", count, time.Now().Format(time.RFC3339))

	// Create blob for new file
	blob := object.NewBlob([]byte(content))
	blobHash, err := g.repo.WriteObject(blob)
	if err != nil {
		return "", fmt.Errorf("writing blob: %w", err)
	}

	// Create new tree with all existing entries plus new file
	tree := object.NewTree()

	// Add existing entries
	for _, entry := range existingEntries {
		tree.AddEntry(entry.Mode, entry.Name, entry.Hash)
	}

	// Add new file
	tree.AddEntry("100644", filename, blobHash)

	treeHash, err := g.repo.WriteObject(tree)
	if err != nil {
		return "", fmt.Errorf("writing tree: %w", err)
	}

	// Create commit
	commitMsg := fmt.Sprintf("Pull #%d at %s", count, time.Now().Format("2006-01-02 15:04:05"))
	commit := object.NewCommit(
		treeHash,
		parentHash,
		"Infinite Git <infinite@example.com>",
		"Infinite Git <infinite@example.com>",
		commitMsg,
	)

	commitHash, err := g.repo.WriteObject(commit)
	if err != nil {
		return "", fmt.Errorf("writing commit: %w", err)
	}

	// Update refs/heads/main
	if err := g.repo.UpdateRef("refs/heads/main", commitHash); err != nil {
		return "", fmt.Errorf("updating ref: %w", err)
	}

	return commitHash, nil
}

// GetCounter returns the current counter value.
func (g *Generator) GetCounter() int64 {
	return atomic.LoadInt64(&g.counter)
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// parseTree parses tree object data into entries.
func parseTree(data []byte) []object.TreeEntry {
	var entries []object.TreeEntry
	i := 0

	for i < len(data) {
		// Find space (end of mode)
		modeEnd := i
		for modeEnd < len(data) && data[modeEnd] != ' ' {
			modeEnd++
		}
		if modeEnd >= len(data) {
			break
		}
		mode := string(data[i:modeEnd])

		// Find null (end of name)
		nameStart := modeEnd + 1
		nameEnd := nameStart
		for nameEnd < len(data) && data[nameEnd] != 0 {
			nameEnd++
		}
		if nameEnd >= len(data) {
			break
		}
		name := string(data[nameStart:nameEnd])

		// Read 20-byte SHA-1
		hashStart := nameEnd + 1
		if hashStart+20 > len(data) {
			break
		}
		hash := fmt.Sprintf("%x", data[hashStart:hashStart+20])

		entries = append(entries, object.TreeEntry{
			Mode: mode,
			Name: name,
			Hash: hash,
		})

		i = hashStart + 20
	}

	return entries
}
