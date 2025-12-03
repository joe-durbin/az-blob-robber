package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/joe-durbin/az-blob-robber/pkg/azure"
	"github.com/joe-durbin/az-blob-robber/pkg/scanner"
	"github.com/joe-durbin/az-blob-robber/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
	flag "github.com/spf13/pflag"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `az-blob-robber - Azure Storage Discovery & Exfiltration Tool

Usage:
  %s [options]

Options:
`, os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Show help
  %s
  %s -h
  %s --help

  # Brute-force scan with default wordlists
  %s -b
  %s --brute-force-defaults

  # Custom wordlists
  %s -A custom-accounts.txt -C custom-containers.txt

  # Target specific account and container
  %s -a mywebsite -c '$webroot'

  # Use with authentication token
  %s -a myaccount -t "eyJ0eXAi..."

  # Adjust concurrency
  %s --concurrency 50

  # Enable debug mode (output curl commands)
  %s -d -b

For more information, visit: https://github.com/joe-durbin/az-blob-robber
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
	}
}

func main() {
	// Define flags with both short and long forms
	bruteForce := flag.BoolP("brute-force-defaults", "b", false, "Use default wordlists for brute-force scanning")
	accountListPath := flag.StringP("accounts", "A", "wordlists/accounts.txt", "Path to file containing storage account names")
	containerListPath := flag.StringP("containers", "C", "wordlists/containers.txt", "Path to file containing container names")
	singleAccount := flag.StringP("account", "a", "", "Single storage account name (skips wordlist)")
	singleContainer := flag.StringP("container", "c", "", "Single container name (skips wordlist)")
	concurrency := flag.IntP("concurrency", "n", 20, "Concurrency level")
	token := flag.StringP("token", "t", "", "Azure Storage access token (optional, for authenticated requests)")
	userAgent := flag.StringP("user-agent", "u", "", "Custom User-Agent string (default: az-blob-robber/1.0)")
	debug := flag.BoolP("debug", "d", false, "Enable debug mode (output curl equivalents for successful requests)")

	flag.Parse()

	// If no arguments provided, or help requested, show usage
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	// Require either brute-force flag or specific account/container
	if !*bruteForce && *singleAccount == "" && *singleContainer == "" {
		fmt.Fprintf(os.Stderr, "Error: Must specify either -b/--brute-force-defaults or provide -a/--account or -c/--container\n\n")
		flag.Usage()
		os.Exit(1)
	}

	var accounts []string
	var containers []string

	// Determine account list
	if *singleAccount != "" {
		accounts = []string{*singleAccount}
	} else if *bruteForce || *singleContainer != "" {
		var err error
		accounts, err = readLines(*accountListPath)
		if err != nil {
			fmt.Printf("Error reading accounts file: %v\n", err)
			os.Exit(1)
		}
		if len(accounts) == 0 {
			fmt.Fprintf(os.Stderr, "Error: No accounts found in wordlist or accounts list is empty\n")
			os.Exit(1)
		}
	}

	// Determine container list
	if *singleContainer != "" {
		containers = []string{*singleContainer}
	} else if *bruteForce || *singleAccount != "" {
		var err error
		containers, err = readLines(*containerListPath)
		if err != nil {
			fmt.Printf("Error reading containers file: %v\n", err)
			os.Exit(1)
		}
		if len(containers) == 0 {
			fmt.Fprintf(os.Stderr, "Error: No containers found in wordlist or containers list is empty\n")
			os.Exit(1)
		}
	}

	// Handle debug file creation
	var debugWriter io.Writer
	if *debug {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("debug_%s.log", timestamp)
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Error creating debug file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		debugWriter = f
		fmt.Printf("Debug mode enabled. Logging to %s\n", filename)
	}

	// Set default user-agent if not provided
	userAgentValue := *userAgent
	if userAgentValue == "" {
		userAgentValue = azure.DefaultUserAgent
	}

	// Initialize Scanner
	s := scanner.NewScanner(accounts, containers, *concurrency, *token, debugWriter, userAgentValue)
	s.Start() // Start background scanning

	// Initialize UI
	model := ui.NewAppModel(s, *token, debugWriter, userAgentValue)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}
