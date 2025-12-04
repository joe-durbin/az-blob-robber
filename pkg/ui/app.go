package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joe-durbin/az-blob-robber/pkg/azure"
	"github.com/joe-durbin/az-blob-robber/pkg/scanner"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sessionState int

const (
	stateScanning sessionState = iota
	stateExploring
)

type AppModel struct {
	state   sessionState
	scanner *scanner.Scanner

	// Lists
	accountList list.Model
	fileList    list.Model

	// Data
	foundItems []scanner.Result // Flat list of Account -> Container
	files      []azure.Blob

	// State
	isScanning     bool
	isLoadingFiles bool

	// Single Download State
	isSingleDownloading       bool
	singleDownloadBlob       *azure.Blob
	singleDownloadPath       string
	singleDownloadBytesRead  int64
	singleDownloadTotalBytes int64
	singleDownloadCancelled  bool
	singleDownloadReader     io.ReadCloser
	singleDownloadWriter     *os.File

	// Modal State
	showModal    bool
	modalType    int // 0: None, 1: Confirm, 2: Alert, 3: Bulk Confirm, 4: Bulk Progress, 5: Single Download Progress
	modalTitle   string
	modalContent string
	pendingBlob  *azure.Blob // For overwrite confirmation

	// UI Components
	spinner spinner.Model

	// Selection
	selectedAccount   string
	selectedContainer string

	// Authentication
	accessToken string
	debugWriter io.Writer // Writer for debug output
	userAgent   string

	// Theme
	currentTheme Theme

	// Layout dimensions
	width  int
	height int

	// Bulk Download State
	isBulkDownloading     bool
	bulkDownloadQueue     []FileItem
	bulkDownloadTotal     int
	bulkDownloadCurrent   int
	bulkCancelRequested   bool
	bulkCurrentBytesRead  int64
	bulkCurrentTotalBytes int64
	bulkDownloadSuccesses int
	bulkDownloadFailures  []string

	// Scan Progress
	scanProgress int
	scanTotal    int
}

func NewAppModel(scanner *scanner.Scanner, token string, debugWriter io.Writer, userAgent string) AppModel {
	// Create lists
	accountList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	accountList.Title = "Accounts & Containers"
	accountList.SetShowStatusBar(false)
	accountList.SetFilteringEnabled(false)

	// Customize account list delegate to be more compact
	accountDelegate := list.NewDefaultDelegate()
	accountDelegate.ShowDescription = false                                                    // Remove description lines for compactness
	accountDelegate.Styles.NormalTitle = accountDelegate.Styles.NormalTitle.Margin(0, 0, 0, 0) // Remove margins
	accountDelegate.Styles.SelectedTitle = accountDelegate.Styles.SelectedTitle.Margin(0, 0, 0, 0)
	accountDelegate.SetSpacing(0) // Remove vertical spacing between items
	accountList.SetDelegate(accountDelegate)

	fileList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	fileList.Title = "Files"
	fileList.SetShowStatusBar(false)
	fileList.SetFilteringEnabled(false)
	fileList.SetShowHelp(false)

	// Customize file list delegate to be more compact
	fileDelegate := list.NewDefaultDelegate()
	fileDelegate.ShowDescription = false                                                 // Remove description lines for compactness
	fileDelegate.Styles.NormalTitle = fileDelegate.Styles.NormalTitle.Margin(0, 0, 0, 0) // Remove margins
	fileDelegate.Styles.SelectedTitle = fileDelegate.Styles.SelectedTitle.Margin(0, 0, 0, 0)
	fileDelegate.SetSpacing(0) // Remove vertical spacing between items
	fileList.SetDelegate(fileDelegate)

	model := AppModel{
		scanner:      scanner,
		accountList:  accountList,
		fileList:     fileList,
		spinner:      spinner.New(),
		state:        stateScanning,
		isScanning:   true,
		accessToken:  token,
		debugWriter:  debugWriter,
		userAgent:    userAgent,
		currentTheme: DefaultTheme,

		isBulkDownloading: false,
		bulkDownloadQueue: []FileItem{},
	}

	// Apply initial theme to lists
	model.accountList, model.fileList = ApplyThemeToLists(model.accountList, model.fileList, DefaultTheme)

	return model
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startScanning(),
	)
}

// Command to start the scanner and listen for results
func (m AppModel) startScanning() tea.Cmd {
	return func() tea.Msg {
		// This is a blocking call if we just iterate, so we need to be careful.
		// Actually, the scanner pushes to a channel. We need a command that waits on the channel.
		// We'll define a separate function for that loop or just a one-off wait.
		// Better pattern: A command that reads one item from the channel and returns it,
		// then re-dispatches itself.
		return <-m.scanner.Results
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Resize lists
		h, v := AppStyle.GetFrameSize()
		m.accountList.SetSize(msg.Width/3-h, msg.Height-v-4)
		m.fileList.SetSize(msg.Width*2/3-h, msg.Height-v-4)

	case tea.KeyMsg:
		if m.showModal {
			switch m.modalType {
			case 1: // Confirm Overwrite
				if msg.String() == "y" || msg.String() == "Y" {
					if m.pendingBlob != nil {
						// Use the new progress-based download system
						blob := *m.pendingBlob
						m.pendingBlob = nil
						
						// Build path (same logic as 'd' key handler)
						dateStr := time.Now().Format("2006-01-02")
						filename := blob.Name
						versionStr := blob.VersionId
						if versionStr == "" {
							versionStr = blob.Snapshot
						}
						
						if versionStr != "" {
							cleanName := filepath.Clean(blob.Name)
							ext := filepath.Ext(cleanName)
							base := cleanName
							if ext != "" {
								base = cleanName[:len(cleanName)-len(ext)]
							}
							t, err := time.Parse(time.RFC3339, versionStr)
							if err == nil {
								ts := t.Format("20060102150405")
								if ext != "" {
									filename = fmt.Sprintf("%s_%s%s", base, ts, ext)
								} else {
									filename = fmt.Sprintf("%s_%s", base, ts)
								}
							} else {
								if ext != "" {
									filename = fmt.Sprintf("%s_%s%s", base, versionStr, ext)
								} else {
									filename = fmt.Sprintf("%s_%s", base, versionStr)
								}
							}
						}
						
						path := filepath.Join("downloads", dateStr, m.selectedAccount, m.selectedContainer, filename)
						
						// Initialize single download state
						m.isSingleDownloading = true
						m.singleDownloadBlob = &blob
						m.singleDownloadPath = path
						m.singleDownloadBytesRead = 0
						m.singleDownloadTotalBytes = blob.Properties.ContentLength
						m.singleDownloadCancelled = false
						
						// Switch to progress modal
						m.modalType = 5 // Single Download Progress
						m.modalTitle = "Downloading File"
						m.modalContent = fmt.Sprintf(
							"Downloading %s...\n\n0%% (0 / %d bytes)\n\nPress ESC to cancel.",
							blob.Name, m.singleDownloadTotalBytes,
						)
						
						return m, m.startSingleDownload(blob, path)
					}
				} else if msg.String() == "n" || msg.String() == "N" || msg.String() == "esc" {
					m.showModal = false
					m.pendingBlob = nil
				}
			case 2: // Alert (Success/Error)
				if msg.String() == "enter" || msg.String() == "esc" {
					m.showModal = false
				}
			case 3: // Bulk download confirm
				if msg.String() == "y" || msg.String() == "Y" {
					m.showModal = true // Keep modal open
					m.modalType = 4    // Switch to progress modal
					m.modalTitle = "Bulk Download in Progress"
					m.modalContent = "Starting download...\n\nPlease wait..."
					return m, m.startBulkDownload()
				} else if msg.String() == "n" || msg.String() == "N" || msg.String() == "esc" {
					m.showModal = false
				}
			case 4: // Bulk download progress
				// Allow cancel with ESC
				if msg.String() == "esc" {
					m.bulkCancelRequested = true
					m.modalTitle = "Cancelling Bulk Download..."
				}
			case 5: // Single download progress
				// Allow cancel with ESC
				if msg.String() == "esc" {
					m.singleDownloadCancelled = true
					m.modalTitle = "Cancelling Download..."
					// Close the reader to unblock any pending Read() calls
					if m.singleDownloadReader != nil {
						m.singleDownloadReader.Close()
						// Don't set to nil yet - let the step function handle cleanup
					}
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			// Toggle focus
			if m.state == stateScanning {
				m.state = stateExploring
			} else {
				m.state = stateScanning
			}

			// Update styles based on focus
			// This is handled in the View method by applying border styles based on m.state.
			// The delegate itself doesn't need to change for focus indication in this setup.
			// if m.state == stateScanning {
			//     m.accountList.SetDelegate(list.NewDefaultDelegate()) // Active
			//     // m.fileList.SetDelegate(inactiveDelegate) // If we had one
			// } else {
			//     // m.accountList.SetDelegate(inactiveDelegate)
			// }

		case "enter":
			// The auto-fetch on selection change handles this now.
			// We can keep 'enter' for other actions if needed, or remove it.
			// For now, let's keep it but it won't trigger file fetching.
			// If we want 'enter' to explicitly switch focus to the right pane, we could add that here.
			// For now, 'tab' handles focus switching.
			// if m.state == stateExploring {
			// 	// Select container and list files
			// 	item, ok := m.accountList.SelectedItem().(ListItem)
			// 	if ok && item.Type == scanner.ResultContainerFound {
			// 		m.selectedAccount = item.AccountName
			// 		m.selectedContainer = item.ContainerName
			// 		return m, m.fetchFiles(item.AccountName, item.ContainerName)
			// 	}
			// }
		case "d":
			if m.state == stateExploring && len(m.files) > 0 {
				// Download selected file (no confirmation; go straight to progress)
				item, ok := m.fileList.SelectedItem().(FileItem)
				if ok {
					dateStr := time.Now().Format("2006-01-02")
					filename := item.Name
					versionStr := item.VersionId
					if versionStr == "" {
						versionStr = item.Snapshot
					}

					if versionStr != "" {
						// Handle versioned filenames, preserving directory structure
						cleanName := filepath.Clean(item.Name)
						ext := filepath.Ext(cleanName)
						base := cleanName
						if ext != "" {
							base = cleanName[:len(cleanName)-len(ext)]
						}
						t, err := time.Parse(time.RFC3339, versionStr)
						if err == nil {
							ts := t.Format("20060102150405")
							if ext != "" {
								filename = fmt.Sprintf("%s_%s%s", base, ts, ext)
							} else {
								filename = fmt.Sprintf("%s_%s", base, ts)
							}
						} else {
							if ext != "" {
								filename = fmt.Sprintf("%s_%s%s", base, versionStr, ext)
							} else {
								filename = fmt.Sprintf("%s_%s", base, versionStr)
							}
						}
					}

					path := filepath.Join("downloads", dateStr, m.selectedAccount, m.selectedContainer, filename)

					// Initialize single download state
					m.isSingleDownloading = true
					m.singleDownloadBlob = &item.Blob
					m.singleDownloadPath = path
					m.singleDownloadBytesRead = 0
					m.singleDownloadTotalBytes = item.Blob.Properties.ContentLength
					m.singleDownloadCancelled = false

					// Show progress modal immediately
					m.showModal = true
					m.modalType = 5 // Single Download Progress
					m.modalTitle = "Downloading File"
					m.modalContent = fmt.Sprintf(
						"Downloading %s...\n\n0%% (0 / %d bytes)\n\nPress ESC to cancel.",
						item.Name, m.singleDownloadTotalBytes,
					)

					return m, m.startSingleDownload(item.Blob, path)
				}
			}

		case "b":
			// Bulk download - download all latest versions in container
			if m.state == stateExploring && len(m.files) > 0 && m.selectedContainer != "" {
				// Show confirmation modal
				m.showModal = true
				m.modalType = 3 // Bulk download confirm
				m.modalTitle = "Bulk Download"
				m.modalContent = fmt.Sprintf("Download all files from %s/%s?\n\nNote: Only latest versions will be downloaded.\nDeleted versioned files will be included.\n\n[y] Yes  [n] No", m.selectedAccount, m.selectedContainer)
				return m, nil
			}
		case "v":
			if m.state == stateExploring && len(m.files) > 0 {
				item, ok := m.fileList.SelectedItem().(FileItem)
				if ok {
					return m, m.fetchVersions(m.selectedAccount, m.selectedContainer, item.Name)
				}
			}
		case "t":
			// Toggle theme
			m.currentTheme = GetNextTheme(m.currentTheme)
			// Apply theme to lists
			m.accountList, m.fileList = ApplyThemeToLists(m.accountList, m.fileList, m.currentTheme)
			return m, nil
		}

	case SingleDownloadStartedMsg:
		if msg.Err != nil {
			// Error starting download
			m.isSingleDownloading = false
			m.showModal = true
			m.modalType = 2
			m.modalTitle = "Download Error"
			m.modalContent = fmt.Sprintf("Failed to start download:\n%s\n\n[Enter] OK", msg.Err.Error())
			return m, nil
		}

		// Store reader and writer in model
		m.singleDownloadReader = msg.Reader
		m.singleDownloadWriter = msg.Writer

		// Start the first download step
		return m, m.singleDownloadStep()

	case SingleDownloadProgressMsg:
		// Update progress in model
		m.singleDownloadBytesRead = msg.BytesRead

		// Calculate percentage
		percent := 0
		if msg.Total > 0 {
			percent = int(float64(msg.BytesRead) / float64(msg.Total) * 100)
		}

		// Update modal content
		if msg.Total > 0 {
			m.modalContent = fmt.Sprintf(
				"Downloading %s...\n\n%d%% (%d / %d bytes)\n\nPress ESC to cancel.",
				msg.BlobName, percent, msg.BytesRead, msg.Total,
			)
		} else {
			m.modalContent = fmt.Sprintf(
				"Downloading %s...\n\n%d bytes downloaded\n\nPress ESC to cancel.",
				msg.BlobName, msg.BytesRead,
			)
		}

		// Check if done or error
		if msg.Done {
			// Clean up resources
			if m.singleDownloadReader != nil {
				m.singleDownloadReader.Close()
				m.singleDownloadReader = nil
			}
			if m.singleDownloadWriter != nil {
				m.singleDownloadWriter.Close()
				m.singleDownloadWriter = nil
			}

			m.isSingleDownloading = false

			if m.singleDownloadCancelled {
				// User cancelled
				m.showModal = true
				m.modalType = 2
				m.modalTitle = "Download Cancelled"
				m.modalContent = fmt.Sprintf("Download of \"%s\" was cancelled.\n\n[Enter] OK", msg.BlobName)
				m.singleDownloadCancelled = false
				m.singleDownloadBlob = nil
				return m, nil
			} else if msg.Err != nil {
				// Error occurred
				m.showModal = true
				m.modalType = 2
				m.modalTitle = "Download Error"
				m.modalContent = fmt.Sprintf("Error downloading file:\n%s\n\n[Enter] OK", msg.Err.Error())
				m.singleDownloadBlob = nil
				return m, nil
			} else {
				// Success
				m.showModal = true
				m.modalType = 2
				m.modalTitle = "Download Complete"
				m.modalContent = fmt.Sprintf("File saved to:\n%s\n\n[Enter] OK", msg.Path)
				m.singleDownloadBlob = nil
				return m, nil
			}
		}

		// Continue downloading (not done yet)
		return m, m.singleDownloadStep()

	case DownloadMsg:
		if msg.Err != nil {
			// Single file download error (legacy path)
			m.showModal = true
			m.modalType = 2
			m.modalTitle = "Download Error"
			m.modalContent = fmt.Sprintf("%s\n\n[Enter] OK", msg.Err.Error())
			return m, nil
		}

	case BulkDownloadProgressMsg:
		// Update byte-level and file-level progress
		m.bulkCurrentBytesRead = msg.Bytes
		m.bulkCurrentTotalBytes = msg.TotalB

		if msg.Err != nil && msg.Done {
			m.bulkDownloadFailures = append(m.bulkDownloadFailures, msg.File)
		} else if msg.Done && msg.Err == nil {
			m.bulkDownloadSuccesses++
		}

		// Calculate percentage for current file if size known
		percent := 0
		if m.bulkCurrentTotalBytes > 0 {
			percent = int(float64(m.bulkCurrentBytesRead) / float64(m.bulkCurrentTotalBytes) * 100)
		}

		m.modalContent = fmt.Sprintf(
			"Downloading files...\n\nFile %d/%d: %s\n%d%% (%d / %d bytes)\n\nPress ESC to cancel.",
			msg.Current, msg.Total, msg.File, percent, m.bulkCurrentBytesRead, m.bulkCurrentTotalBytes,
		)

		// If this file is done, either move to next or finish
		if msg.Done {
			// If cancellation requested or we've reached the end, show summary
			if m.bulkCancelRequested || msg.Current >= msg.Total {
				m.isBulkDownloading = false

				// Build summary
				remaining := msg.Total - msg.Current
				var summary string
				if m.bulkCancelRequested {
					summary = fmt.Sprintf("Bulk download cancelled.\n\nDownloaded: %d\nFailed: %d\nRemaining skipped: %d",
						m.bulkDownloadSuccesses, len(m.bulkDownloadFailures), remaining)
					m.modalTitle = "Bulk Download Cancelled"
				} else if len(m.bulkDownloadFailures) > 0 {
					summary = fmt.Sprintf("Downloaded: %d\nFailed: %d\n\nFailed files:\n%s",
						m.bulkDownloadSuccesses, len(m.bulkDownloadFailures), strings.Join(m.bulkDownloadFailures, "\n"))
					m.modalTitle = "Bulk Download Finished with Errors"
				} else {
					summary = fmt.Sprintf("Successfully downloaded %d files!", m.bulkDownloadSuccesses)
					m.modalTitle = "Bulk Download Complete!"
				}

				m.modalType = 2 // Alert (OK button)
				m.modalContent = summary + "\n\n[Enter] OK"
				return m, nil
			}

			// Continue with next file
			return m, m.downloadNextFile()
		}

		// Continue downloading current file
		return m, m.bulkDownloadStep()

	case scanner.Result:
		if msg.Type == scanner.ResultScanFinished {
			m.isScanning = false
			m.state = stateScanning
			return m, nil
		}

		if msg.Type == scanner.ResultProgressUpdate {
			m.scanProgress = msg.Progress
			m.scanTotal = msg.Total
			return m, m.startScanning()
		}

		if msg.Type == scanner.ResultAccountFound {
			return m, m.startScanning()
		}

		if msg.Type == scanner.ResultContainerFound {
			// Only show PUBLIC containers
			if !msg.IsPublic {
				return m, m.startScanning()
			}

			// Check if we already have the account header in foundItems
			accountHeaderExists := false
			for _, item := range m.foundItems {
				if item.Type == scanner.ResultAccountFound && item.AccountName == msg.AccountName {
					accountHeaderExists = true
					break
				}
			}

			if !accountHeaderExists {
				// Add account header first
				m.foundItems = append(m.foundItems, scanner.Result{
					Type:        scanner.ResultAccountFound,
					AccountName: msg.AccountName,
				})
			}

			// Add container
			m.foundItems = append(m.foundItems, msg)

			// Update list model
			items := make([]list.Item, len(m.foundItems))
			for i, res := range m.foundItems {
				items[i] = ListItem{Result: res}
			}
			m.accountList.SetItems(items)
		}

		// Continue listening
		return m, m.startScanning()

	case FileListMsg:
		m.isLoadingFiles = false
		m.files = msg.Blobs
		items := make([]list.Item, len(m.files))
		for i, f := range m.files {
			items[i] = FileItem{Blob: f}
		}
		m.fileList.SetItems(items)

	case VersionsMsg:
		if msg.Err != nil {
			m.showModal = true
			m.modalType = 2
			m.modalTitle = "Error"
			m.modalContent = fmt.Sprintf("Failed to fetch versions:\n%v\n\n[Enter] OK", msg.Err)
			return m, nil
		}

		// Find the parent item index
		var parentIdx int = -1
		items := m.fileList.Items()

		for i, it := range items {
			if f, ok := it.(FileItem); ok && f.Name == msg.BlobName && !f.IsVersion {
				parentIdx = i
				break
			}
		}

		if parentIdx != -1 {
			// Check if already expanded (simple check: is next item a version of this?)
			// Or we can track expansion state in a map or in the item (but item is value type).
			// We'll just check if next item is a version.
			isExpanded := false
			if parentIdx+1 < len(items) {
				if next, ok := items[parentIdx+1].(FileItem); ok && next.IsVersion && next.Name == msg.BlobName {
					isExpanded = true
				}
			}

			newItems := make([]list.Item, 0)
			newItems = append(newItems, items[:parentIdx+1]...) // Add up to parent

			if !isExpanded {
				// Insert versions
				for _, v := range msg.Versions {
					// Skip the current version (which is the parent) if it has no snapshot/versionId
					if v.Snapshot == "" && v.VersionId == "" {
						continue // Skip base blob
					}
					newItems = append(newItems, FileItem{Blob: v, IsVersion: true})
				}
				// Add rest
				if parentIdx+1 < len(items) {
					newItems = append(newItems, items[parentIdx+1:]...)
				}
			} else {
				// Collapse: Remove versions
				// Skip items until we hit a non-version or different name
				restIdx := parentIdx + 1
				for restIdx < len(items) {
					if next, ok := items[restIdx].(FileItem); ok && next.IsVersion && next.Name == msg.BlobName {
						restIdx++
					} else {
						break
					}
				}
				if restIdx < len(items) {
					newItems = append(newItems, items[restIdx:]...)
				}
			}

			m.fileList.SetItems(newItems)
		}
		return m, nil
	}

	// Update components
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	// Handle List Updates and Auto-Fetch
	if m.state == stateScanning { // Left Pane Focused
		prevSelected := m.accountList.SelectedItem()
		m.accountList, cmd = m.accountList.Update(msg)
		cmds = append(cmds, cmd)

		currSelected := m.accountList.SelectedItem()

		// Check if selection changed
		if currSelected != nil && (prevSelected == nil || prevSelected.FilterValue() != currSelected.FilterValue()) {
			item, ok := currSelected.(ListItem)
			if ok && item.Type == scanner.ResultContainerFound {
				m.selectedAccount = item.AccountName
				m.selectedContainer = item.ContainerName
				m.isLoadingFiles = true
				m.files = []azure.Blob{} // Clear current files
				m.fileList.SetItems([]list.Item{})
				cmds = append(cmds, m.fetchFiles(item.AccountName, item.ContainerName))
			}
		}
	} else { // Right Pane Focused
		m.fileList, cmd = m.fileList.Update(msg)
		cmds = append(cmds, cmd)
		// We still want to see the left pane, but maybe not update it?
		// Actually we should probably let it update but not handle keys?
		// Bubble Tea lists handle keys if you call Update.
		// So we only call Update on the focused list.
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Header
	header := GetHeaderStyle(m.currentTheme).Render("az-blob-robber - Azure Storage Explorer")

	// Main Content
	var content string

	// Left Pane: Accounts/Containers
	leftStyle := GetPaneStyle().Width(m.width / 3)
	rightStyle := GetPaneStyle().Width(m.width * 2 / 3)

	if m.state == stateScanning {
		leftStyle = leftStyle.BorderForeground(m.currentTheme.FocusedBorder)
		rightStyle = rightStyle.BorderForeground(m.currentTheme.UnfocusedBorder)
	} else { // stateExploring
		leftStyle = leftStyle.BorderForeground(m.currentTheme.UnfocusedBorder)
		rightStyle = rightStyle.BorderForeground(m.currentTheme.FocusedBorder)
	}
	leftPane := leftStyle.Render(m.accountList.View())

	// Right Pane: Files or Welcome
	var rightPaneContent string
	if m.isLoadingFiles {
		loadingStyle := lipgloss.NewStyle().Foreground(m.currentTheme.NormalText)
		rightPaneContent = loadingStyle.Render(fmt.Sprintf("Loading files for %s/%s...", m.selectedAccount, m.selectedContainer))
	} else if len(m.files) > 0 || m.selectedContainer != "" {
		// Check if versioning is enabled (any file has VersionId)
		hasVersioning := false
		for _, f := range m.files {
			if f.VersionId != "" || f.Snapshot != "" {
				hasVersioning = true
				break
			}
		}

		// Build the file list view
		fileListView := m.fileList.View()

		// If versioning detected, prepend a notice
		if hasVersioning {
			versionNotice := lipgloss.NewStyle().
				Bold(true).
				Foreground(m.currentTheme.Accent).
				Background(m.currentTheme.SubtleBackground).
				Padding(0, 1).
				Render("🔄 VERSIONING ENABLED • Press 'v' to expand history")
			rightPaneContent = versionNotice + "\n" + fileListView
		} else {
			rightPaneContent = fileListView
		}
	} else {
		// Style the welcome message
		welcomeStyle := lipgloss.NewStyle().Foreground(m.currentTheme.NormalText)
		rightPaneContent = welcomeStyle.Render("Select a container to view files.")
	}

	rightPane := rightStyle.Render(rightPaneContent)

	content = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Footer / Status
	var statusIcon string
	if m.isScanning {
		spinnerView := m.spinner.View()
		if m.scanTotal > 0 {
			statusIcon = fmt.Sprintf("%s %d/%d", spinnerView, m.scanProgress, m.scanTotal)
		} else {
			statusIcon = spinnerView
		}
	} else {
		statusIcon = "Done"
	}

	var hints string
	if m.state == stateScanning {
		hints = "Tab: Focus Files | ↑/↓: Navigate Accounts"
	} else {
		hints = "Tab: Focus Accounts | 'd': Download | 'v': Versions | 'b': Bulk Download"
	}

	// Count only containers (not accounts)
	containerCount := 0
	for _, item := range m.foundItems {
		if item.Type == scanner.ResultContainerFound {
			containerCount++
		}
	}
	statusStyle := lipgloss.NewStyle().Foreground(m.currentTheme.StatusText)
	status := statusStyle.Render(fmt.Sprintf("Found: %d | Status: %s | %s | 'q': Quit", containerCount, statusIcon, hints))

	baseView := lipgloss.JoinVertical(lipgloss.Left, header, content, status)

	if m.showModal {
		// Modal Style
		modalStyle := lipgloss.NewStyle().
			Width(50).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.currentTheme.FocusedBorder).
			Padding(1, 2).
			Align(lipgloss.Left)

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(m.currentTheme.Accent).MarginBottom(1)

		modalView := modalStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				titleStyle.Render(m.modalTitle),
				m.modalContent,
			),
		)

		// Center modal on screen
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalView,
			lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(m.currentTheme.SubText))
	}

	return baseView
}

// --- Helper Types & Commands ---

type ListItem struct {
	scanner.Result
}

func (i ListItem) Title() string {
	if i.Type == scanner.ResultAccountFound {
		return "▶ " + i.AccountName
	}
	// Container
	return "  └─ " + i.ContainerName
}

func (i ListItem) Description() string {
	// Return empty to save vertical space
	return ""
}

func (i ListItem) FilterValue() string {
	if i.Type == scanner.ResultAccountFound {
		return i.AccountName
	}
	return i.ContainerName
}

type FileItem struct {
	azure.Blob
	IsVersion bool
	Expanded  bool
}

func (i FileItem) Title() string {
	// Choose icon based on file type
	icon := "📄"

	// Check if this file has versioning (VersionId or Snapshot indicates versioning is enabled)
	hasVersioning := i.VersionId != "" || i.Snapshot != ""

	// Prioritize deleted status - makes deleted files very obvious
	if i.Blob.IsDeleted() {
		icon = "🗑️" // Deleted icon (trash can)
	} else if hasVersioning {
		icon = "🔄" // Versioned file icon
	}

	// Format size in human-readable format
	size := formatSize(i.Properties.ContentLength)

	// Build the display string
	var displayName string
	if i.IsVersion {
		// For versions, show the version timestamp indented
		version := i.VersionId
		if version == "" {
			version = i.Snapshot
		}
		displayName = fmt.Sprintf("  └─ %s %s", version, size)
	} else {
		// Regular file: icon + name + size on the right
		displayName = fmt.Sprintf("%s %s %s%s", icon, i.Name,
			padRight("", 1), // Add spacing
			size)
	}

	// Add deleted marker if applicable
	if i.Blob.IsDeleted() && !i.IsVersion {
		displayName += " (Deleted)"
	}

	return displayName
}

func (i FileItem) Description() string {
	// Keep description minimal to save vertical space
	// Only show last modified for non-versions
	if !i.IsVersion && i.Properties.LastModified != "" {
		return "  " + i.Properties.LastModified
	}
	return ""
}

// formatSize converts bytes to human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// padRight pads a string to a minimum width
func padRight(s string, minWidth int) string {
	if len(s) >= minWidth {
		return s
	}
	return s + " "
}
func (i FileItem) FilterValue() string { return i.Name }

type FileListMsg struct {
	Blobs []azure.Blob
}

// Messages for single download with progress
type SingleDownloadStartedMsg struct {
	Reader   io.ReadCloser
	Writer   *os.File
	Path     string
	BlobName string
	Err      error
}

type SingleDownloadProgressMsg struct {
	BytesRead int64
	Total     int64
	Path      string
	BlobName  string
	Done      bool
	Err       error
}

type DownloadMsg struct {
	Err     error
	Path    string
	Success bool   // For bulk downloads
	Message string // For bulk download summary
}

type BulkDownloadProgressMsg struct {
	Current int
	Total   int
	File    string
	Bytes   int64
	TotalB  int64
	Done    bool
	Err     error
}

// --- Single Download Commands ---

// startSingleDownload begins a single file download by opening the HTTP stream and file.
func (m AppModel) startSingleDownload(blob azure.Blob, path string) tea.Cmd {
	return func() tea.Msg {
		c := azure.NewClientWithToken(m.accessToken, m.debugWriter, m.userAgent)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return SingleDownloadStartedMsg{Err: err, BlobName: blob.Name}
		}

		// Determine which identifier to use (VersionId takes precedence over Snapshot)
		identifier := blob.VersionId
		if identifier == "" {
			identifier = blob.Snapshot
		}

		rc, err := c.DownloadBlob(m.selectedAccount, m.selectedContainer, blob.Name, identifier)
		if err != nil {
			return SingleDownloadStartedMsg{Err: err, BlobName: blob.Name}
		}

		f, err := os.Create(path)
		if err != nil {
			rc.Close()
			return SingleDownloadStartedMsg{Err: err, BlobName: blob.Name}
		}

		return SingleDownloadStartedMsg{
			Reader:   rc,
			Writer:   f,
			Path:     path,
			BlobName: blob.Name,
			Err:      nil,
		}
	}
}

// singleDownloadStep reads a chunk from the current download and reports progress.
// Note: This function needs to access the model's reader/writer, so it must be called
// after SingleDownloadStartedMsg has been processed and the reader/writer are set.
func (m AppModel) singleDownloadStep() tea.Cmd {
	return func() tea.Msg {
		// Access the current model's reader/writer (these are set when SingleDownloadStartedMsg is received)
		reader := m.singleDownloadReader
		writer := m.singleDownloadWriter
		currentBytes := m.singleDownloadBytesRead
		totalBytes := m.singleDownloadTotalBytes
		path := m.singleDownloadPath
		blobName := ""
		if m.singleDownloadBlob != nil {
			blobName = m.singleDownloadBlob.Name
		}

		if reader == nil || writer == nil {
			return SingleDownloadProgressMsg{
				BytesRead: currentBytes,
				Total:     totalBytes,
				Path:      path,
				BlobName:  blobName,
				Done:      true,
				Err:       fmt.Errorf("download not properly initialized"),
			}
		}

		// Check if user requested cancel (read from model)
		if m.singleDownloadCancelled {
			return SingleDownloadProgressMsg{
				BytesRead: currentBytes,
				Total:     totalBytes,
				Path:      path,
				BlobName:  blobName,
				Done:      true,
				Err:       nil,
			}
		}

		// Read a chunk
		buf := make([]byte, 32*1024)
		n, err := reader.Read(buf)

		newBytesRead := currentBytes
		if n > 0 {
			if _, werr := writer.Write(buf[:n]); werr != nil && err == nil {
				err = werr
			}
			newBytesRead = currentBytes + int64(n)
		}

		done := false
		var finalErr error

		if err != nil {
			if err == io.EOF {
				done = true
				finalErr = nil
			} else {
				done = true
				finalErr = err
			}
		}

		return SingleDownloadProgressMsg{
			BytesRead: newBytesRead,
			Total:     totalBytes,
			Path:      path,
			BlobName:  blobName,
			Done:      done,
			Err:       finalErr,
		}
	}
}

// --- Bulk Download Commands (with per-file progress) ---

func (m AppModel) fetchFiles(account, container string) tea.Cmd {
	return func() tea.Msg {
		c := azure.NewClientWithToken(m.accessToken, m.debugWriter, m.userAgent)
		blobs, err := c.ListBlobs(account, container)
		if err != nil {
			// Return empty list on error - UI will show empty file list
			// Error is logged via debug writer if enabled
			return FileListMsg{Blobs: []azure.Blob{}}
		}
		return FileListMsg{Blobs: blobs}
	}
}

type VersionsMsg struct {
	BlobName string
	Versions []azure.Blob
	Err      error
}

func (m AppModel) fetchVersions(account, container, blobName string) tea.Cmd {
	return func() tea.Msg {
		c := azure.NewClientWithToken(m.accessToken, m.debugWriter, m.userAgent)
		versions, err := c.GetBlobVersions(account, container, blobName)
		return VersionsMsg{BlobName: blobName, Versions: versions, Err: err}
	}
}

func (m AppModel) downloadFile(blob azure.Blob, overwrite bool) tea.Cmd {
	return func() tea.Msg {
		c := azure.NewClientWithToken(m.accessToken, m.debugWriter, m.userAgent)

		// Structure: downloads/date/account/container/file
		dateStr := time.Now().Format("2006-01-02")
		baseDir := filepath.Join("downloads", dateStr, m.selectedAccount, m.selectedContainer)

		// Clean the file path to prevent traversal
		cleanName := filepath.Clean(blob.Name)
		if strings.Contains(cleanName, "..") {
			return DownloadMsg{Err: fmt.Errorf("invalid filename: %s", blob.Name)}
		}

		path := filepath.Join(baseDir, cleanName)

		versionStr := blob.VersionId
		if versionStr == "" {
			versionStr = blob.Snapshot
		}

		if versionStr != "" {
			// Append version timestamp to filename, preserving directory structure
			// Use cleanName to ensure we're working with a safe path
			ext := filepath.Ext(cleanName)
			base := cleanName
			if ext != "" {
				base = cleanName[:len(cleanName)-len(ext)]
			}

			// Parse and reformat
			t, err := time.Parse(time.RFC3339, versionStr)
			if err == nil {
				ts := t.Format("20060102150405")
				if ext != "" {
					path = filepath.Join(baseDir, fmt.Sprintf("%s_%s%s", base, ts, ext))
				} else {
					path = filepath.Join(baseDir, fmt.Sprintf("%s_%s", base, ts))
				}
			} else {
				// Fallback if parse fails
				if ext != "" {
					path = filepath.Join(baseDir, fmt.Sprintf("%s_%s%s", base, versionStr, ext))
				} else {
					path = filepath.Join(baseDir, fmt.Sprintf("%s_%s", base, versionStr))
				}
			}
		}

		// If not overwrite and exists, we should have caught it in Update.
		// But double check or just proceed if overwrite is true.

		// Determine which identifier to use (VersionId takes precedence over Snapshot)
		identifier := blob.VersionId
		if identifier == "" {
			identifier = blob.Snapshot
		}

		rc, err := c.DownloadBlob(m.selectedAccount, m.selectedContainer, blob.Name, identifier)
		if err != nil {
			return DownloadMsg{Err: err}
		}
		defer rc.Close()

		// Create file
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return DownloadMsg{Err: err}
		}
		f, err := os.Create(path)
		if err != nil {
			return DownloadMsg{Err: err}
		}
		defer f.Close()

		_, err = io.Copy(f, rc)
		if err != nil {
			return DownloadMsg{Err: err}
		}

		return DownloadMsg{Err: nil, Path: path}
	}
}

// startBulkDownload initializes the bulk download process
func (m *AppModel) startBulkDownload() tea.Cmd {
	// Reset state
	m.isBulkDownloading = true
	m.bulkCancelRequested = false
	m.bulkCurrentBytesRead = 0
	m.bulkCurrentTotalBytes = 0
	m.bulkDownloadSuccesses = 0
	m.bulkDownloadFailures = []string{}
	m.bulkDownloadQueue = []FileItem{}

	// Filter files to download (non-version entries only)
	for _, item := range m.fileList.Items() {
		fileItem := item.(FileItem)
		// Skip version entries (indented history items under parent)
		if !fileItem.IsVersion {
			m.bulkDownloadQueue = append(m.bulkDownloadQueue, fileItem)
		}
	}

	m.bulkDownloadTotal = len(m.bulkDownloadQueue)
	m.bulkDownloadCurrent = 0

	if m.bulkDownloadTotal == 0 {
		m.isBulkDownloading = false
		// Immediately show a simple alert when there are no files
		return func() tea.Msg { return BulkDownloadProgressMsg{Current: 0, Total: 0, File: "", Bytes: 0, TotalB: 0, Done: true, Err: nil} }
	}

	// Start first download
	return m.downloadNextFile()
}

// downloadNextFile prepares the next file in the queue and opens its streams
func (m *AppModel) downloadNextFile() tea.Cmd {
	if len(m.bulkDownloadQueue) == 0 {
		return nil
	}

	fileItem := m.bulkDownloadQueue[0]
	m.bulkDownloadQueue = m.bulkDownloadQueue[1:] // Dequeue
	m.bulkDownloadCurrent++

	return func() tea.Msg {
		// Download this file (open streams, then step via bulkDownloadStep)
		c := azure.NewClientWithToken(m.accessToken, m.debugWriter, m.userAgent)

		// Structure: downloads/date/account/container/file
		dateStr := time.Now().Format("2006-01-02")
		baseDir := filepath.Join("downloads", dateStr, m.selectedAccount, m.selectedContainer)

		// Clean the file path to prevent traversal
		cleanName := filepath.Clean(fileItem.Name)
		if strings.Contains(cleanName, "..") {
			return BulkDownloadProgressMsg{
				Current: m.bulkDownloadCurrent,
				Total:   m.bulkDownloadTotal,
				File:    fileItem.Name,
				Bytes:   0,
				TotalB:  0,
				Done:    true,
				Err:     fmt.Errorf("invalid filename: %s", fileItem.Name),
			}
		}

		path := filepath.Join(baseDir, cleanName)

		// Ensure parent directory exists (handle nested files like static/js/foo.js)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return BulkDownloadProgressMsg{
				Current: m.bulkDownloadCurrent,
				Total:   m.bulkDownloadTotal,
				File:    fileItem.Name,
				Bytes:   0,
				TotalB:  0,
				Done:    true,
				Err:     err,
			}
		}

		// Determine which identifier to use
		identifier := fileItem.VersionId
		if identifier == "" {
			identifier = fileItem.Snapshot
		}

		rc, err := c.DownloadBlob(m.selectedAccount, m.selectedContainer, fileItem.Name, identifier)
		if err != nil {
			return BulkDownloadProgressMsg{
				Current: m.bulkDownloadCurrent,
				Total:   m.bulkDownloadTotal,
				File:    fileItem.Name,
				Bytes:   0,
				TotalB:  0,
				Done:    true,
				Err:     err,
			}
		}

		outFile, err := os.Create(path)
		if err != nil {
			rc.Close()
			return BulkDownloadProgressMsg{
				Current: m.bulkDownloadCurrent,
				Total:   m.bulkDownloadTotal,
				File:    fileItem.Name,
				Bytes:   0,
				TotalB:  0,
				Done:    true,
				Err:     err,
			}
		}

		// Initialize current-byte tracking
		m.bulkCurrentBytesRead = 0
		m.bulkCurrentTotalBytes = fileItem.Properties.ContentLength

		// Store reader/writer temporarily via a stepping message
		// We'll embed them in the next step through a helper closure.
		return m.bulkDownloadStepWithHandles(fileItem, rc, outFile)
	}
}

// bulkDownloadStepWithHandles reads a chunk for the given file and reports progress.
func (m *AppModel) bulkDownloadStepWithHandles(fileItem FileItem, rc io.ReadCloser, outFile *os.File) BulkDownloadProgressMsg {
	if m.bulkCancelRequested {
		rc.Close()
		outFile.Close()
		return BulkDownloadProgressMsg{
			Current: m.bulkDownloadCurrent,
			Total:   m.bulkDownloadTotal,
			File:    fileItem.Name,
			Bytes:   m.bulkCurrentBytesRead,
			TotalB:  m.bulkCurrentTotalBytes,
			Done:    true,
			Err:     nil,
		}
	}

	buf := make([]byte, 32*1024)
	n, err := rc.Read(buf)
	if n > 0 {
		if _, werr := outFile.Write(buf[:n]); werr != nil && err == nil {
			err = werr
		}
		m.bulkCurrentBytesRead += int64(n)
	}

	done := false
	if err != nil {
		if err == io.EOF {
			done = true
			err = nil
		} else {
			done = true
		}
	}

	if done {
		rc.Close()
		outFile.Close()
	}

	return BulkDownloadProgressMsg{
		Current: m.bulkDownloadCurrent,
		Total:   m.bulkDownloadTotal,
		File:    fileItem.Name,
		Bytes:   m.bulkCurrentBytesRead,
		TotalB:  m.bulkCurrentTotalBytes,
		Done:    done,
		Err:     err,
	}
}

// bulkDownloadStep continues downloading the current file using the last known state.
func (m *AppModel) bulkDownloadStep() tea.Cmd {
	// We don't retain rc/outFile on the model, so this is only used immediately
	// after downloadNextFile, which wrapped them into a BulkDownloadProgressMsg.
	// Here we just no-op and rely on downloadNextFile's step handling.
	return nil
}
