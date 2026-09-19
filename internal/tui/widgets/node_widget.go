package widgets

import (
	"charm.land/lipgloss/v2"
	"github.com/micros/microsctl/internal/network"
	"image/color"
)

func NodeColors(node network.Device, selected bool) (color.Color, color.Color) {
	border, name := ColorBorder, ColorWarn
	if node.Online {
		name = ColorOnline
	}
	if selected {
		border = ColorAccent
	}
	if node.SpecialEndpoint() != "" {
		border, name = lipgloss.Color("#75BFFF"), lipgloss.Color("#75BFFF")
		if selected {
			border = lipgloss.Color("#C6E4FF")
		}
	}
	return border, name
}

func NodeTitle(node network.Device) string {
	if node.SpecialEndpoint() == "Localhost" {
		return "localhost"
	}
	name := Clean(node.Name)
	if endpoint := node.SpecialEndpoint(); endpoint != "" {
		if name == "" {
			return endpoint
		}
		return endpoint + " · " + name
	}
	return name
}
