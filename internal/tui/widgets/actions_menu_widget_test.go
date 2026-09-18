package widgets

import (
	"strings"
	"testing"

	"github.com/micros/microsctl/internal/usb"
)

func TestDiscoveryDisplaysFlashCapacityInsteadOfRawID(t *testing.T) {
	for _, tc := range []struct{ size, id, want string }{
		{"4 MB", "20:4016", "4 MB"},
		{"", "00:8B95", "Unknown"},
	} {
		state := State{Inventory: usb.Inventory{Devices: []usb.Device{{Info: &usb.DeviceInfo{FlashSize: tc.size, FlashID: tc.id}}}}}
		got := state.discoveryDetails(80)
		if !strings.Contains(got, tc.want) || strings.Contains(got, tc.id) {
			t.Fatalf("flash display must show %q without raw ID %q: %s", tc.want, tc.id, got)
		}
	}
}
