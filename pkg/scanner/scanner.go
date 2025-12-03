package scanner

import (
	"io"
	"sync"

	"github.com/joe-durbin/az-blob-robber/pkg/azure"
)

type ResultType int

const (
	ResultAccountFound ResultType = iota
	ResultContainerFound
	ResultScanFinished
	ResultProgressUpdate
)

type Result struct {
	Type          ResultType
	AccountName   string
	ContainerName string
	IsPublic      bool
	Progress      int // Current progress count
	Total         int // Total combinations to test
}

type Scanner struct {
	AccountNames   []string
	ContainerNames []string
	Results        chan Result
	Concurrency    int
	Token          string
	DebugWriter    io.Writer
}

func NewScanner(accountNames, containerNames []string, concurrency int, token string, debugWriter io.Writer) *Scanner {
	return &Scanner{
		AccountNames:   accountNames,
		ContainerNames: containerNames,
		Results:        make(chan Result, 100),
		Concurrency:    concurrency,
		Token:          token,
		DebugWriter:    debugWriter,
	}
}

func (s *Scanner) Start() {
	go func() {
		defer close(s.Results)

		client := azure.NewClientWithToken(s.Token, s.DebugWriter)

		// Calculate total combinations
		total := len(s.AccountNames) * len(s.ContainerNames)
		
		// Track progress with mutex
		var mu sync.Mutex
		progress := 0

		// Helper to send progress update
		sendProgress := func() {
			mu.Lock()
			current := progress
			mu.Unlock()
			s.Results <- Result{
				Type:     ResultProgressUpdate,
				Progress: current,
				Total:    total,
			}
		}

		var wg sync.WaitGroup
		sem := make(chan struct{}, s.Concurrency)

		// If we have accounts to brute force
		for _, acc := range s.AccountNames {
			wg.Add(1)
			sem <- struct{}{} // Acquire token

			go func(accountName string) {
				defer wg.Done()
				defer func() { <-sem }() // Release token

				// 1. Check if account exists
				mu.Lock()
				progress++
				mu.Unlock()
				sendProgress()

				if client.CheckAccount(accountName) {
					s.Results <- Result{
						Type:        ResultAccountFound,
						AccountName: accountName,
					}

					// 2. If account exists, brute force containers
					s.scanContainers(accountName, client, &mu, &progress, sendProgress)
				}
			}(acc)
		}

		wg.Wait()
		s.Results <- Result{Type: ResultScanFinished}
	}()
}

func (s *Scanner) scanContainers(accountName string, client *azure.Client, mu *sync.Mutex, progress *int, sendProgress func()) {
	for _, cont := range s.ContainerNames {
		// Check sequentially per account
		mu.Lock()
		*progress++
		mu.Unlock()
		sendProgress()

		exists, public := client.CheckContainer(accountName, cont)
		if exists {
			s.Results <- Result{
				Type:          ResultContainerFound,
				AccountName:   accountName,
				ContainerName: cont,
				IsPublic:      public,
			}
		}
	}
}
