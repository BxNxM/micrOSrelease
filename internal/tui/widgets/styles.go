package widgets

import "charm.land/lipgloss/v2"

var (
	ColorAccent  = lipgloss.Color("#68E0C1")
	ColorText    = lipgloss.Color("#E6EDF3")
	ColorMuted   = lipgloss.Color("#7D8997")
	ColorBorder  = lipgloss.Color("#394554")
	ColorWarn    = lipgloss.Color("#FFCC66")
	ColorError   = lipgloss.Color("#FF6B7A")
	ColorOnline  = lipgloss.Color("#65D98B")
	ColorRelease = lipgloss.Color("#75BFFF")
	ColorOff     = ColorMuted

	StyleLogo     = lipgloss.NewStyle().Bold(true).Foreground(ColorText).Background(lipgloss.Color("#24564E")).Padding(0, 1)
	StyleTitle    = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	StyleMuted    = lipgloss.NewStyle().Foreground(ColorMuted)
	StylePanel    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorBorder).Padding(1, 2)
	StyleSelected = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	StyleKey      = lipgloss.NewStyle().Bold(true).Foreground(ColorText).Background(ColorBorder).Padding(0, 1)
)
