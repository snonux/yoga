package app

import "github.com/charmbracelet/lipgloss"

var (
	videoExtensions = map[string]struct{}{
		".mp4": {},
		".mkv": {},
		".mov": {},
		".avi": {},
		".wmv": {},
		".m4v": {},
		".webm": {},
	}
	tableStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1)
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	filterStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("105")).Padding(1, 2)
	statusStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
)
