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

> flake -attempts 5000
Running 'go test -race -count=1 -v ./...' up to 5000 times (use -h for help)
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
