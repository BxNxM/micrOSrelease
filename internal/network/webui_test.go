package network

import "testing"

func TestWebUIURL(t *testing.T) {
	for _, tc := range []struct{ name, feature, want string }{
		{"Entrance", "ON", "http://Entrance.local"},
		{"Entrance", "OFF", ""},
		{"Entrance", "", ""},
		{"node-1", "ON", "http://node-1.local"},
		{"", "ON", ""},
		{"bad/name", "ON", ""},
		{"node.local@evil", "ON", ""},
		{"-node", "ON", ""},
	} {
		t.Run(tc.name+tc.feature, func(t *testing.T) {
			d := Device{Name: tc.name, Features: map[string]string{"webui": tc.feature}}
			if got := d.WebUIURL(); got != tc.want {
				t.Fatalf("WebUIURL() = %q, want %q", got, tc.want)
			}
		})
	}
}
