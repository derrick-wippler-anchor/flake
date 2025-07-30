package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var attempts int
	var help bool
	flag.IntVar(&attempts, "attempts", 100, "Maximum number of test attempts")
	flag.BoolVar(&help, "h", false, "Show help")
	flag.Parse()

	if help {
		fmt.Println("flake - Run Go tests repeatedly until they fail")
		fmt.Println("Usage: flake [options]")
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	fmt.Printf("Running 'go test -race -count=1 -v ./...' up to %d times (use -h for help)\n", attempts)

	for i := 1; i <= attempts; i++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			fmt.Printf("\nInterrupted after %d attempts\n", i-1)
			os.Exit(0)
		default:
		}

		cmd := exec.CommandContext(ctx, "go", "test", "-race", "-count=1", "-v", "./...")

		// Start spinner
		done := make(chan bool)
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

		output, err := cmd.CombinedOutput()
		done <- true
		fmt.Print(" \b")

		if err != nil {
			// Check if this is due to context cancellation (Ctrl+C)
			if ctx.Err() != nil {
				fmt.Printf("\nInterrupted after %d attempts\n", i-1)
				os.Exit(0)
			}
			// Test failed - print full output and exit
			fmt.Printf("\nTest failed on attempt %d:\n", i)
			fmt.Print(string(output))
			fmt.Printf("\033[31mTest failed after %d attempts\033[0m\n", i)
			os.Exit(1)
		}

		// Test passed - print dot without newline
		fmt.Print(".")
	}

	// All attempts completed successfully
	fmt.Printf("\nAll %d test attempts passed successfully!\n", attempts)
}