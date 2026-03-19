package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Banner styles per environment
	BannerDevStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#00FF00")).Padding(0, 1)
	BannerStageStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#FFFF00")).Padding(0, 1)
	BannerProdStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#FF0000")).Padding(0, 1)

	// Step status styles
	StepOKStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00CC00")).Bold(true)
	StepFailStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3333")).Bold(true)
	StepWarnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")).Bold(true)
	StepInfoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#524ff7"))

	// Environment label styles (foreground only, for inline text)
	EnvDevStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF00"))
	EnvStageStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFF00"))
	EnvProdStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3333"))

	// Summary styles
	LabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	ValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)

	// Warning style
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true)
)
