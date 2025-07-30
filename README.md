# flake

A CLI tool to run Go tests repeatedly until they fail, useful for detecting flaky tests.

- Runs `go test -v -race -count=1 ./...` repeatedly
- Print a dot (`.`) for each passing test run
- Show the full test output when a test fails
- Can be interrupted with Ctrl+C

## WHY?

1. Running `go test -count=100 ./...` runs the tests 100 times, but all
   iterations in the same process with the same database container, but when
   tests run separately (like in CI or when run individually), they might expose
   timing issues.
2. Database transaction timing - There might be edge cases where the transaction
   handling or database connection pooling affects the idempotency logic which
   is only caught when running the `go test ./...` repeatedly from the command
   line.

## Installation

```bash
# To install from github
go install github.com/derrick-wippler-anchor/flake@latest

# Or checkout the code and run
make install
```

### Options
- `-h`: Display help message
- `-attempts <number>`: Maximum number of test attempts (default: 100)

### Examples

```bash
> flake
Running 'go test -race -count=1 -v ./...' up to 100 times (use -h for help)
.........^C 
Interrupted after 9 attempts

> flake -attempts 50
Running 'go test -race -count=1 -v ./...' up to 50 times (use -h for help)
..................................................

> flake
Running 'go test -race -count=1 -v ./...' up to 100 times (use -h for help)
................................
Test failed on attempt 33:
=== RUN   TestSleep
    main_test.go:13: Random test failure (10% chance)
--- FAIL: TestSleep (1.00s)
FAIL
FAIL    flake   1.010s
FAIL
Test failed after 33 attempts
```

## MCP Server Mode

Flake can run as an MCP (Model Context Protocol) server, allowing agentic clients like Claude Code to use it as a tool to detect flaky tests in any Go project directory.

### Usage

Start flake in MCP server mode:

```bash
flake -mcp
```

This starts an MCP server that communicates over stdio, making it compatible with Claude Code and other MCP clients.

### MCP Tool: `run_flake_tests`

The MCP server provides a single tool:

**Parameters:**
- `directory` (required): Working directory where tests should be run
- `attempts` (optional): Maximum number of test attempts (default: 100)

**Returns:**
- Success/failure status
- Number of attempts made
- Full test output (on failure)
- Error messages if applicable