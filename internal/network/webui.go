package network

import "strings"

// WebUIURL uses the reachable address for special endpoints and mDNS for LAN nodes.
func (d Device) WebUIURL() string {
	if !strings.EqualFold(d.Features["webui"], "ON") {
		return ""
	}
	switch d.SpecialEndpoint() {
	case "Localhost":
		return "http://localhost"
	case "AP mode":
		return "http://192.168.4.1"
	}
	if len(d.Name) == 0 || len(d.Name) > 63 {
		return ""
	}
	for i, c := range d.Name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}
		if c != '-' || i == 0 || i == len(d.Name)-1 {
			return ""
		}
	}
	return "http://" + d.Name + ".local"
}
