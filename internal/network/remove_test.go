package network

import (
	"errors"
	"testing"
)

type removalStore struct {
	nodes []Device
	err   error
}

func (s *removalStore) LoadDevices() ([]Device, error) { return s.nodes, nil }
func (s *removalStore) SaveDevices(nodes []Device) error {
	if s.err != nil {
		return s.err
	}
	s.nodes = nodes
	return nil
}

func TestRemoveDevice(t *testing.T) {
	store := &removalStore{nodes: []Device{{UID: "one", Address: "a"}, {UID: "two", Address: "b"}}}
	s := &Service{Store: store}
	if _, err := s.Cached(); err != nil {
		t.Fatal(err)
	}
	store.err = errors.New("write failed")
	if err := s.RemoveDevice("one"); err == nil || len(s.known) != 2 {
		t.Fatal("failed persistence must retain the node")
	}
	store.err = nil
	if err := s.RemoveDevice("one"); err != nil {
		t.Fatal(err)
	}
	if len(s.known) != 1 || len(store.nodes) != 1 || store.nodes[0].UID != "two" {
		t.Fatal("removal must update memory and persistent cache, preserving other nodes")
	}
}
