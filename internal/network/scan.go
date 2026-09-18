package network

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"sync"
)

// Service discovers authenticated micrOS nodes and retains known nodes across
// refreshes so disconnected devices remain visible as offline cards.
type Service struct {
	CIDR     string
	Password string
	mu       sync.Mutex
	known    map[string]Device
	Store    DeviceStore
	inspect  func(context.Context, Device, string) (Device, bool)
}

// Cached loads saved observations without opening network connections.
func (s *Service) Cached() ([]Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Store == nil {
		return nil, nil
	}
	nodes, err := s.Store.LoadDevices()
	if err != nil {
		return nil, err
	}
	nodes = append([]Device(nil), nodes...)
	s.known = make(map[string]Device, len(nodes))
	var cached []Device
	for i := range nodes {
		nodes[i].Cached = true
		s.known[nodes[i].Address] = nodes[i]
	}
	for _, node := range UniqueDevices(nodes) {
		// Keep special identities for alias matching, but never display them
		// without a live check.
		if node.SpecialEndpoint() != "" {
			continue
		}
		cached = append(cached, node)
	}
	return cached, nil
}

func (s *Service) Discover(ctx context.Context, emit func(Device)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefixes, err := scanPrefixes(s.CIDR)
	if err != nil {
		return err
	}
	targets := map[string]Device{}
	for _, prefix := range prefixes {
		prefix = prefix.Masked()
		for ip := prefix.Addr(); prefix.Contains(ip); ip = ip.Next() {
			if prefix.Bits() < 31 && (ip == prefix.Addr() || !prefix.Contains(ip.Next())) {
				continue
			}
			address := net.JoinHostPort(ip.String(), "9008")
			targets[address] = Device{Address: address}
		}
	}
	if s.known == nil {
		s.known = make(map[string]Device)
	}
	for address, node := range s.known {
		targets[address] = node
	}
	// Always check these endpoints, even outside the LAN CIDR or without Wi-Fi.
	for _, node := range specialTargets() {
		if _, known := targets[node.Address]; !known {
			targets[node.Address] = node
		}
	}
	probe := s.inspect
	if probe == nil {
		probe = inspect
	}
	jobs := make(chan Device)
	results := make(chan Device, 32)
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range jobs {
				node, valid := probe(ctx, target, s.Password)
				if target.SpecialEndpoint() != "" || valid || target.UID != "" {
					select {
					case results <- node:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		// Check the two special entry points first, then saved nodes and the LAN.
		for _, target := range specialTargets() {
			select {
			case jobs <- targets[target.Address]:
			case <-ctx.Done():
				return
			}
		}
		for _, knownFirst := range []bool{true, false} {
			for _, target := range targets {
				if target.SpecialEndpoint() != "" || (target.UID != "") != knownFirst {
					continue
				}
				select {
				case jobs <- target:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	go func() { wg.Wait(); close(results) }()
	for node := range results {
		s.known[node.Address] = node
		emit(node)
	}
	if s.Store != nil {
		var nodes []Device
		for _, node := range s.known {
			nodes = append(nodes, node)
		}
		nodes = UniqueDevices(nodes)
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
		if err := s.Store.SaveDevices(nodes); err != nil {
			return fmt.Errorf("save device cache: %w", err)
		}
	}
	return ctx.Err()
}

func scanPrefixes(cidr string) ([]netip.Prefix, error) {
	if cidr != "" {
		p, err := netip.ParsePrefix(cidr)
		if err != nil || !p.Addr().Is4() || p.Bits() < 24 {
			return nil, fmt.Errorf("scan CIDR must be IPv4 /24 or smaller")
		}
		return []netip.Prefix{p.Masked()}, nil
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	seen := map[netip.Prefix]bool{}
	var prefixes []netip.Prefix
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			p, err := netip.ParsePrefix(address.String())
			if err != nil || !p.Addr().Is4() || !p.Addr().IsPrivate() {
				continue
			}
			p = netip.PrefixFrom(p.Addr(), max(24, p.Bits())).Masked()
			if !seen[p] {
				prefixes = append(prefixes, p)
				seen[p] = true
			}
		}
	}
	return prefixes, nil
}
