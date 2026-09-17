package tui

import (
	"context"
	"testing"

	"github.com/micros/micros-release/internal/network"
)

type gatedDiscovery struct{ release chan struct{} }

func (d gatedDiscovery) Discover(ctx context.Context, emit func(network.Device)) error {
	emit(network.Device{UID: "first", Name: "First"})
	select {
	case <-d.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	emit(network.Device{UID: "second", Name: "Second"})
	return nil
}

func TestDiscoveryDeliversBeforeCompletion(t *testing.T) {
	d := gatedDiscovery{release: make(chan struct{})}
	m := New(nil, d)
	m.discovering = true
	command := m.networkCmd()
	defer m.scanCancel()
	first := command().(networkMsg)
	if first.device == nil || first.device.UID != "first" || first.done {
		t.Fatal("expected first result before scan completion")
	}
	updated, next := m.Update(first)
	m = updated.(model)
	if len(m.nodes) != 1 || !m.discovering {
		t.Fatal("first card must be visible during discovery")
	}
	close(d.release)
	updated, next = m.Update(next())
	m = updated.(model)
	if len(m.nodes) != 2 || m.nodeIndex != 0 {
		t.Fatal("new results must preserve selection")
	}
	updated, _ = m.Update(next())
	if updated.(model).discovering {
		t.Fatal("scan must finish")
	}
}
