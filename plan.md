# Infinite Git HTTP Server Implementation Plan

## Overview
I'll implement a Go HTTP server that generates a new commit every time a client pulls from the repository. This creates an "infinite" Git repository where the `main` branch is updated with a new commit on each fetch operation.

## Key Components

### 1. HTTP Endpoints
- `GET /info/refs?service=git-upload-pack` - Reference discovery (triggers commit generation)
- `POST /git-upload-pack` - Handle fetch/clone with dynamically generated commits
- Reject all push operations with 403 Forbidden

### 2. Core Modules
- `pktline` package - Handle Git's packet line format
- `server` package - HTTP server and routing
- `generator` package - Generate new commits on demand
- `upload` package - Modified git-upload-pack to serve dynamic content
- `repo` package - Git repository management

### 3. Commit Generation Strategy
- On each pull request, generate a new commit with:
  - Timestamp in commit message
  - Unique content (e.g., pull counter, random data, or timestamp)
  - Parent pointing to previous HEAD of main
- Update main branch to point to new commit
- Use Git plumbing commands or go-git library for commit creation

### 4. Project Structure
```
.
├── cmd/
│   └── infinite-git/
│       └── main.go          # Entry point
├── internal/
│   ├── pktline/
│   │   ├── reader.go        # Pkt-line reader
│   │   └── writer.go        # Pkt-line writer
│   ├── server/
│   │   ├── server.go        # HTTP server
│   │   └── handlers.go      # HTTP handlers
│   ├── generator/
│   │   └── commit.go        # Commit generation logic
│   ├── upload/
│   │   └── pack.go          # Modified git-upload-pack
│   └── repo/
│       └── repo.go          # Repository management
├── go.mod
└── README.md
```

### 5. Implementation Details
- Initialize a bare Git repository on server startup
- Intercept ref discovery requests to generate new commits
- Use mutex to handle concurrent pull requests safely
- Generate commits using either:
  - `git hash-object`, `git update-index`, `git write-tree`, `git commit-tree`
  - Or go-git library for pure Go implementation
- Each commit could contain:
  - A file with incrementing counter
  - Timestamp of the pull request
  - Client information (if available)

### 6. Features
- Thread-safe commit generation
- Persistent Git repository on disk
- Configurable commit content generation
- Rate limiting (optional)
- Logging of all pull operations
- Environment variable configuration

## Example Behavior
```bash
# First client pull
$ git pull origin main
# Gets commit 1a2b3c4... "Pull #1 at 2024-01-20 10:00:00"

# Second client pull (moments later)
$ git pull origin main  
# Gets commit 5d6e7f8... "Pull #2 at 2024-01-20 10:00:05"
```