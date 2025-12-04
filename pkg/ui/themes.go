package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Theme holds color definitions for the TUI
type Theme struct {
	Name             string
	FocusedBorder    lipgloss.Color
	UnfocusedBorder  lipgloss.Color
	ModalBorder      lipgloss.Color
	ModalBackground  lipgloss.Color
	HintText         lipgloss.Color
	SuccessText      lipgloss.Color
	ErrorText        lipgloss.Color
	Accent           lipgloss.Color // For highlights and notices
	SubtleBackground lipgloss.Color // For background elements
	HeaderForeground lipgloss.Color // Header text color
	HeaderBackground lipgloss.Color // Header background color
	SubText          lipgloss.Color // Subtle text color
	NormalText       lipgloss.Color // Normal list item text
	SelectedText     lipgloss.Color // Selected list item text
	SelectedBg       lipgloss.Color // Selected list item background
	TitleText        lipgloss.Color // List title text
	StatusText       lipgloss.Color // Status bar text
}

// DefaultTheme is the original color scheme
var DefaultTheme = Theme{
	Name:             "Default",
	FocusedBorder:    lipgloss.Color("69"),  // Blue
	UnfocusedBorder:  lipgloss.Color("240"), // Gray
	ModalBorder:      lipgloss.Color("205"), // Pink
	ModalBackground:  lipgloss.Color("235"), // Dark gray
	HintText:         lipgloss.Color("241"), // Light gray
	SuccessText:      lipgloss.Color("42"),  // Green
	ErrorText:        lipgloss.Color("196"), // Red
	Accent:           lipgloss.Color("69"),  // Blue
	SubtleBackground: lipgloss.Color("238"), // Subtle gray
	HeaderForeground: lipgloss.Color("#FAFAFA"),
	HeaderBackground: lipgloss.Color("#7D56F4"), // Purple
	SubText:          lipgloss.Color("#767676"),
	NormalText:       lipgloss.Color("#FAFAFA"),
	SelectedText:     lipgloss.Color("#04B575"), // Green
	SelectedBg:       lipgloss.Color("#3C3C3C"),
	TitleText:        lipgloss.Color("#FAFAFA"),
	StatusText:       lipgloss.Color("#FAFAFA"),
}

// RosePineTheme is a Rose Pine inspired color scheme
var RosePineTheme = Theme{
	Name:             "Rose Pine",
	FocusedBorder:    lipgloss.Color("203"), // Love (rose pink)
	UnfocusedBorder:  lipgloss.Color("244"), // Muted
	ModalBorder:      lipgloss.Color("249"), // Subtle
	ModalBackground:  lipgloss.Color("233"), // Base
	HintText:         lipgloss.Color("245"), // Muted
	SuccessText:      lipgloss.Color("108"), // Foam (teal)
	ErrorText:        lipgloss.Color("210"), // Love (soft red)
	Accent:           lipgloss.Color("180"), // Pine (blue-green)
	SubtleBackground: lipgloss.Color("234"), // Surface
	HeaderForeground: lipgloss.Color("#FAFAFA"),
	HeaderBackground: lipgloss.Color("203"), // Rose pink
	SubText:          lipgloss.Color("245"), // Muted
	NormalText:       lipgloss.Color("#e0def4"),
	SelectedText:     lipgloss.Color("#ebbcba"),
	SelectedBg:       lipgloss.Color("234"),
	TitleText:        lipgloss.Color("#e0def4"),
	StatusText:       lipgloss.Color("#e0def4"),
}

// HackerTheme is a classic green-on-black hacker aesthetic
var HackerTheme = Theme{
	Name:             "Hacker",
	FocusedBorder:    lipgloss.Color("#00FF00"), // Bright green
	UnfocusedBorder:  lipgloss.Color("#008800"), // Dark green
	ModalBorder:      lipgloss.Color("#00FF00"), // Bright green
	ModalBackground:  lipgloss.Color("#000000"), // Black
	HintText:         lipgloss.Color("#00AA00"), // Medium green
	SuccessText:      lipgloss.Color("#00FF00"), // Bright green
	ErrorText:        lipgloss.Color("#FF0000"), // Red (for contrast)
	Accent:           lipgloss.Color("#00FF00"), // Bright green
	SubtleBackground: lipgloss.Color("#001100"), // Very dark green
	HeaderForeground: lipgloss.Color("#00FF00"), // Bright green
	HeaderBackground: lipgloss.Color("#000000"), // Black
	SubText:          lipgloss.Color("#00AA00"), // Medium green
	NormalText:       lipgloss.Color("#00FF00"), // Bright green
	SelectedText:     lipgloss.Color("#000000"), // Black (on green bg)
	SelectedBg:       lipgloss.Color("#00FF00"), // Bright green
	TitleText:        lipgloss.Color("#00FF00"), // Bright green
	StatusText:       lipgloss.Color("#00FF00"), // Bright green
}

// CatppuccinTheme is inspired by Catppuccin Mocha color palette
var CatppuccinTheme = Theme{
	Name:             "Catppuccin",
	FocusedBorder:    lipgloss.Color("#89b4fa"), // Blue
	UnfocusedBorder:  lipgloss.Color("#45475a"), // Gray
	ModalBorder:      lipgloss.Color("#cba6f7"), // Mauve
	ModalBackground:  lipgloss.Color("#313244"), // Surface0
	HintText:         lipgloss.Color("#bac2de"), // Subtext1
	SuccessText:      lipgloss.Color("#a6e3a1"), // Green
	ErrorText:        lipgloss.Color("#f38ba8"), // Red
	Accent:           lipgloss.Color("#89b4fa"), // Blue
	SubtleBackground: lipgloss.Color("#1e1e2e"), // Base
	HeaderForeground: lipgloss.Color("#cdd6f4"), // Text
	HeaderBackground: lipgloss.Color("#89b4fa"), // Blue
	SubText:          lipgloss.Color("#6c7086"), // Subtext0
	NormalText:       lipgloss.Color("#cdd6f4"), // Text
	SelectedText:     lipgloss.Color("#1e1e2e"), // Base (on blue bg)
	SelectedBg:       lipgloss.Color("#89b4fa"), // Blue
	TitleText:        lipgloss.Color("#cdd6f4"), // Text
	StatusText:       lipgloss.Color("#cdd6f4"), // Text
}

// VibrantTheme is a super colorful theme with vibrant background
var VibrantTheme = Theme{
	Name:             "Vibrant",
	FocusedBorder:    lipgloss.Color("#FF00FF"), // Magenta
	UnfocusedBorder:  lipgloss.Color("#00FFFF"), // Cyan
	ModalBorder:      lipgloss.Color("#FFFF00"), // Yellow
	ModalBackground:  lipgloss.Color("#1a0033"), // Deep purple
	HintText:         lipgloss.Color("#FFB6C1"), // Light pink
	SuccessText:      lipgloss.Color("#00FF00"), // Bright green
	ErrorText:        lipgloss.Color("#FF1493"), // Deep pink
	Accent:           lipgloss.Color("#FF00FF"), // Magenta
	SubtleBackground: lipgloss.Color("#330033"), // Dark purple
	HeaderForeground: lipgloss.Color("#FFFFFF"), // White
	HeaderBackground: lipgloss.Color("#FF00FF"), // Magenta
	SubText:          lipgloss.Color("#FFB6C1"), // Light pink
	NormalText:       lipgloss.Color("#FFFFFF"), // White
	SelectedText:     lipgloss.Color("#000000"), // Black (on magenta bg)
	SelectedBg:       lipgloss.Color("#FF00FF"), // Magenta
	TitleText:        lipgloss.Color("#FFFF00"), // Yellow
	StatusText:       lipgloss.Color("#00FFFF"), // Cyan
}

var themes = []Theme{DefaultTheme, RosePineTheme, HackerTheme, CatppuccinTheme, VibrantTheme}

// GetNextTheme cycles to the next theme
func GetNextTheme(current Theme) Theme {
	for i, t := range themes {
		if t.Name == current.Name {
			return themes[(i+1)%len(themes)]
		}
	}
	return DefaultTheme
}

// AppStyle is a simple utility style for frame sizing (theme-independent)
var AppStyle = lipgloss.NewStyle().Margin(1, 2)

// GetHeaderStyle returns a theme-aware header style
func GetHeaderStyle(theme Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(theme.HeaderForeground).
		Background(theme.HeaderBackground).
		Padding(0, 1).
		Bold(true)
}

// GetPaneStyle returns a base pane style (border colors are set dynamically)
func GetPaneStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
}

// ApplyThemeToLists updates list delegate styles to match the current theme
// Returns updated account and file lists
func ApplyThemeToLists(accountList, fileList list.Model, theme Theme) (list.Model, list.Model) {
	// Update account list delegate
	accountDelegate := list.NewDefaultDelegate()
	accountDelegate.ShowDescription = false
	accountDelegate.Styles.NormalTitle = accountDelegate.Styles.NormalTitle.
		Foreground(theme.NormalText).
		Margin(0, 0, 0, 0)
	accountDelegate.Styles.SelectedTitle = accountDelegate.Styles.SelectedTitle.
		Foreground(theme.SelectedText).
		Background(theme.SelectedBg).
		Margin(0, 0, 0, 0)
	accountDelegate.SetSpacing(0)
	accountList.SetDelegate(accountDelegate)
	accountList.Styles.Title = accountList.Styles.Title.Foreground(theme.TitleText)

	// Update file list delegate
	fileDelegate := list.NewDefaultDelegate()
	fileDelegate.ShowDescription = false
	fileDelegate.Styles.NormalTitle = fileDelegate.Styles.NormalTitle.
		Foreground(theme.NormalText).
		Margin(0, 0, 0, 0)
	fileDelegate.Styles.SelectedTitle = fileDelegate.Styles.SelectedTitle.
		Foreground(theme.SelectedText).
		Background(theme.SelectedBg).
		Margin(0, 0, 0, 0)
	fileDelegate.SetSpacing(0)
	fileList.SetDelegate(fileDelegate)
	fileList.Styles.Title = fileList.Styles.Title.Foreground(theme.TitleText)

	return accountList, fileList
}

