package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	var attempts int
	var help bool
	var mcp bool
	flag.IntVar(&attempts, "attempts", 100, "Maximum number of test attempts")
	flag.BoolVar(&help, "h", false, "Show help")
	flag.BoolVar(&mcp, "mcp", false, "Run as MCP server for agentic clients")
	flag.Parse()

	if help {
		fmt.Println("flake - Run Go tests repeatedly until they fail")
		fmt.Println("Usage: flake [options]")
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if mcp {
		runMCPServer()
		return
	}

	result := runFlakeTests(".", attempts, true)
	if result.Failed {
		os.Exit(1)
	}
}

type FlakeResult struct {
	Failed       bool
	Attempts     int
	Output       string
	Interrupted  bool
	ErrorMessage string
}

func runFlakeTests(directory string, attempts int, interactive bool) FlakeResult {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if interactive {
		fmt.Printf("Running 'go test -race -count=1 -v ./...' up to %d times (use -h for help)\n", attempts)
	}

	for i := 1; i <= attempts; i++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			msg := fmt.Sprintf("Interrupted after %d attempts", i-1)
			if interactive {
				fmt.Printf("\n%s\n", msg)
			}
			return FlakeResult{
				Failed:      false,
				Attempts:    i - 1,
				Interrupted: true,
				Output:      msg,
			}
		default:
		}

		cmd := exec.CommandContext(ctx, "go", "test", "-race", "-count=1", "-v", "./...")
		cmd.Dir = directory

		var done chan bool
		if interactive {
			// Start spinner
			done = make(chan bool)
			go func() {
				spinner := []string{"|", "/", "-", "\\"}
				for {
					select {
					case <-done:
						return
					default:
						for _, char := range spinner {
							select {
							case <-done:
								return
							default:
								fmt.Print(char)
								fmt.Print("\b")
								time.Sleep(50 * time.Millisecond)
							}
						}
					}
				}
			}()
		}

		output, err := cmd.CombinedOutput()

		if interactive && done != nil {
			done <- true
			fmt.Print(" \b")
		}

		if err != nil {
			// Check if this is due to context cancellation (Ctrl+C)
			if ctx.Err() != nil {
				message := fmt.Sprintf("Interrupted after %d attempts", i-1)
				if interactive {
					fmt.Printf("\n%s\n", message)
				}
				return FlakeResult{
					Failed:      false,
					Attempts:    i - 1,
					Interrupted: true,
					Output:      message,
				}
			}
			// Test failed
			msg := fmt.Sprintf("Test failed on attempt %d", i)
			if interactive {
				fmt.Printf("\n%s:\n", msg)
				fmt.Print(string(output))
				fmt.Printf("\033[31mTest failed after %d attempts\033[0m\n", i)
			}
			return FlakeResult{
				Failed:       true,
				Attempts:     i,
				Output:       string(output),
				ErrorMessage: msg,
			}
		}

		// Test passed - print dot without newline
		if interactive {
			fmt.Print(".")
		}
	}

	// All attempts completed successfully
	msg := fmt.Sprintf("All %d test attempts passed successfully!", attempts)
	if interactive {
		fmt.Printf("\n%s\n", msg)
	}
	return FlakeResult{
		Failed:   false,
		Attempts: attempts,
		Output:   msg,
	}
}

type FlakeTestParams struct {
	Directory string `json:"directory" jsonschema:"required,description=Working directory where tests should be run"`
	Attempts  int    `json:"attempts,omitempty" jsonschema:"description=Maximum number of test attempts (default: 100)"`
}

type FlakeTestResult struct {
	Success      bool   `json:"success"`
	Attempts     int    `json:"attempts"`
	Output       string `json:"output"`
	Interrupted  bool   `json:"interrupted,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func runMCPServer() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "flake",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "run_flake_tests",
		Description: "Run Go tests repeatedly using flake to detect flaky tests in any directory",
	}, runFlakeTestsTool)

	if err := server.Run(context.Background(), mcp.NewStdioTransport()); err != nil {
		log.Fatal(err)
	}
}

func runFlakeTestsTool(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[FlakeTestParams]) (*mcp.CallToolResultFor[FlakeTestResult], error) {
	// Set default attempts if not provided
	attempts := params.Arguments.Attempts
	if attempts <= 0 {
		attempts = 100
	}

	// Validate and clean directory path
	directory := strings.TrimSpace(params.Arguments.Directory)
	if directory == "" {
		return &mcp.CallToolResultFor[FlakeTestResult]{
			Content: []mcp.Content{&mcp.TextContent{
				Text: "Error: directory parameter is required",
			}},
		}, nil
	}

	// Clean and validate the directory path
	cleanDir, err := filepath.Abs(directory)
	if err != nil {
		return &mcp.CallToolResultFor[FlakeTestResult]{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Error: invalid directory path: %v", err),
			}},
		}, nil
	}

	// Check if directory exists
	if _, err := os.Stat(cleanDir); os.IsNotExist(err) {
		return &mcp.CallToolResultFor[FlakeTestResult]{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Error: directory does not exist: %s", cleanDir),
			}},
		}, nil
	}

	// Run flake tests
	result := runFlakeTests(cleanDir, attempts, false)

	var message string
	if result.Failed {
		message = fmt.Sprintf("Tests failed after %d attempts in directory: %s\n\nOutput:\n%s\n\nError: %s",
			result.Attempts, cleanDir, result.Output, result.ErrorMessage)
	} else if result.Interrupted {
		message = fmt.Sprintf("Tests interrupted after %d attempts in directory: %s", result.Attempts, cleanDir)
	} else {
		message = fmt.Sprintf("All %d test attempts passed successfully in directory: %s", result.Attempts, cleanDir)
	}

	return &mcp.CallToolResultFor[FlakeTestResult]{
		Content: []mcp.Content{&mcp.TextContent{
			Text: message,
		}},
	}, nil
}
