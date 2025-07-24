# Infinite Git

A Git HTTP server that generates a new commit every time someone pulls from the repository.

## Features

- Implements Git smart HTTP protocol for read-only access
- Generates a unique commit on every pull/clone operation
- Thread-safe commit generation
- Persistent Git repository on disk
- Structured logging with slog

## Installation

```bash
go install github.com/imjasonh/infinite-git/cmd/infinite-git@latest
```

Or build from source:

```bash
git clone https://github.com/imjasonh/infinite-git
cd infinite-git
go build -o infinite-git ./cmd/infinite-git
```

## Usage

Start the server:

```bash
infinite-git
```

Options:
- `-addr`: HTTP server address (default: ":8080")
- `-repo`: Path to Git repository (default: "./infinite-repo")
- `-log-level`: Log level - debug, info, warn, error (default: "info")

Example:
```bash
infinite-git -addr :3000 -repo /tmp/my-infinite-repo -log-level debug
```

## Testing

Clone the repository:
```bash
git clone http://localhost:8080
```

Every time you pull, you'll get a new commit:
```bash
cd <cloned-repo>
git pull origin main
# New commit appears!
```

## How It Works

1. When a client initiates a pull/clone, the server intercepts the reference discovery request
2. Before advertising refs, it generates a new commit with:
   - A unique file containing the pull counter and timestamp
   - A commit message indicating when the pull occurred
3. The new commit is added to the main branch
4. The updated refs are sent to the client
5. The client receives the new commit as part of the normal Git protocol flow

## Implementation Details

- Uses Git plumbing commands (`git add`, `git commit`) to create commits
- Implements pkt-line format for Git protocol communication
- Delegates object transfer to `git upload-pack` for efficiency
- Rejects all push attempts with 403 Forbidden

## License

MIT