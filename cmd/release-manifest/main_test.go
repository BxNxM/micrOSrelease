package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	selfupdate "github.com/micros/microsctl/internal/network/self_update"
	"gopkg.in/yaml.v3"
)

func TestManifestMatchesMetadata(t *testing.T) {
	output := filepath.Join(t.TempDir(), "MANIFEST.yaml")
	if err := generate("../..", output); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	checkedIn, err := os.ReadFile("../../MANIFEST.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, checkedIn) {
		t.Fatal("MANIFEST.yaml is stale; run make manifest")
	}
	var metadata map[string]any
	if err := yaml.Unmarshal(generated, &metadata); err != nil {
		t.Fatal(err)
	}
	micros, ok := metadata["micros"].(map[string]any)
	if !ok || len(micros) != 1 || micros["version"] == "" {
		t.Fatalf("micros must contain only the latest version: %v", metadata["micros"])
	}
}

func TestManifestPreservesVersionWithoutInspectingBinaries(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "MANIFEST.yaml")
	if err := os.WriteFile(output, []byte(`schema: 1
microsctl:
  version: 9.8.7
  url: https://downloads.example.com/custom-branch/
`), 0644); err != nil {
		t.Fatal(err)
	}
	firmware := filepath.Join(root, "storage/frameworks/esp32")
	if err := os.MkdirAll(firmware, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(firmware, "micrOS-esp32-1.28.0-3.6.3-0.bin"), []byte("metadata only"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(firmware, "micrOS-esp32-1.28.0-3.6.2-0.bin"), []byte("older metadata"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generate(root, output); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(output)
	manifest, err := selfupdate.ParseReleaseManifest(data)
	if err != nil || manifest.Microsctl.Version != "9.8.7" || manifest.MicrOS.Version != "3.6.3-0" || len(manifest.Microsctl.Binaries) != 4 {
		t.Fatalf("manifest=%+v err=%v", manifest, err)
	}
	if manifest.Microsctl.URL != "https://downloads.example.com/custom-branch/" {
		t.Fatal("generation overwrote configured release URLs")
	}
}
