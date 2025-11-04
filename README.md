# TaskFlow

A simple (for now :p) CLI task management system built in Go

## Installation

### Prerequisites

- Go 1.21 or higher
- SQLite3

### Build from Source

```bash
# Clone the repo
git clone https://github.com/sanchxt/golang-task-management.git
cd golang-task-management

# Install dependencies
go mod tidy

# Build
go build -o taskflow ./cmd/taskflow/

# Optional: Move to PATH
sudo mv taskflow /usr/local/bin/
```

## Usage

### Add a Task

```bash
# Simple task
taskflow add "Fix login bug"

# Task with all options
taskflow add "Implement user authentication" \
  --priority high \
  --description "Add JWT-based authentication system" \
  --project "Backend API" \
  --tags auth,security \
  --due-date "2025-12-31"
```

### Command Options

- `-p, --priority`: Set priority (low, medium, high, urgent) - default: medium
- `-d, --description`: Add detailed description
- `-P, --project`: Assign to a project
- `-t, --tags`: Add comma-separated tags
- `--due-date`: Set due date (YYYY-MM-DD format)

### Get Help

```bash
taskflow --help
taskflow add --help
```

## Development

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test -v ./internal/domain/
go test -v ./internal/repository/sqlite/
```

### Database Location

Tasks are stored in `~/.taskflow/tasks.db`

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
