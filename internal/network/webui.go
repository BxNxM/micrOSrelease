package network

import "strings"

// WebUIURL returns the node's mDNS URL only for an enabled Web UI and a safe hostname.
func (d Device) WebUIURL() string {
	if !strings.EqualFold(d.Features["webui"], "ON") || len(d.Name) == 0 || len(d.Name) > 63 {
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
