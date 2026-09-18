package network

import (
	"net"
	"strings"
)

const (
	LocalhostAddress = "127.0.0.1:9008"
	APAddress        = "192.168.4.1:9008"
)

// SpecialEndpoint labels the two transient entry points.
func (d Device) SpecialEndpoint() string {
	host, _, err := net.SplitHostPort(d.Address)
	if err != nil {
		host = d.Address
	}
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return "Localhost"
	case "192.168.4.1":
		return "AP mode"
	default:
		return ""
	}
}

// CardKey identifies a physical device across endpoints. Failed probes without
// an identity are tracked by address until a successful hello supplies the UID.
func (d Device) CardKey() string {
	if d.UID != "" {
		return d.UID
	}
	return "address:" + d.Address
}

// UniqueDevices chooses one observation per UID. Reachable endpoints win; among
// equally available endpoints, prefer localhost, then AP, then an ordinary LAN
// address. Retaining unavailable special observations prevents cached LAN aliases
// from resurrecting an offline simulator.
func UniqueDevices(nodes []Device) []Device {
	var unique []Device
	indexes := make(map[string]int)
	priority := func(d Device) int {
		rank := 2
		switch d.SpecialEndpoint() {
		case "Localhost":
			rank = 0
		case "AP mode":
			rank = 1
		}
		if !d.Online || d.Cached {
			rank += 3
		}
		return rank
	}
	for _, node := range nodes {
		key := node.CardKey()
		if i, ok := indexes[key]; ok {
			old := unique[i]
			if priority(node) < priority(old) || (priority(node) == priority(old) &&
				(node.CheckedAt.After(old.CheckedAt) || (node.CheckedAt.Equal(old.CheckedAt) && node.Address < old.Address))) {
				unique[i] = node
			}
		} else {
			indexes[key] = len(unique)
			unique = append(unique, node)
		}
	}
	return unique
}

func specialTargets() []Device {
	return []Device{{Address: LocalhostAddress}, {Address: APAddress}}
}
