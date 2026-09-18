package usb

import (
	"reflect"
	"testing"
	"testing/fstest"

	assets "github.com/micros/microsctl/storage"
)

func TestBundledResourcesSelectPlatformIO(t *testing.T) {
	resources, err := bundledResources(assets.Files(), "esp32c3")
	if err != nil {
		t.Fatal(err)
	}
	foundSelectedIO, foundOtherIO, foundWeb := false, false, false
	for _, resource := range resources {
		switch resource.Target {
		case "/modules/IO_esp32c3.mpy":
			foundSelectedIO = true
		case "/web/index.html":
			foundWeb = true
		}
		if resource.Target == "/modules/IO_esp32.mpy" || resource.Target == "/modules/IO_esp32s3.mpy" {
			foundOtherIO = true
		}
	}
	if !foundSelectedIO || !foundWeb || foundOtherIO {
		t.Fatalf("unexpected bundled resource selection: selectedIO=%v web=%v otherIO=%v", foundSelectedIO, foundWeb, foundOtherIO)
	}
}

func TestResourcesPreserveHierarchyAndOverrides(t *testing.T) {
	files := fstest.MapFS{
		"modules/README.md":                {Data: []byte("documentation")},
		"modules/main.py":                  {Data: []byte("startup")},
		"modules/modules/feature.py":       {Data: []byte("common")},
		"modules/modules/IO_esp32.py":      {Data: []byte("other board")},
		"modules/modules/IO_esp32c3.py":    {Data: []byte("selected board")},
		"modules/web/assets/IO_chart.js":   {Data: []byte("script")},
		"modules/data/nested/default.json": {Data: []byte("{}")},
		"modules/overrides/feature.py":     {Data: []byte("override")},
	}
	resources, err := prepareResources(files, InstallConfig{Platform: "esp32c3", REPL: REPLConfig{
		Resources: []UpdateResource{{Source: "modules/overrides/feature.py", Target: "/modules/feature.py"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]string)
	for _, resource := range resources {
		got[resource.target] = string(resource.data)
	}
	want := map[string]string{
		"/main.py": "startup", "/modules/feature.py": "override",
		"/modules/IO_esp32c3.py": "selected board", "/web/assets/IO_chart.js": "script",
		"/data/nested/default.json": "{}", "/overrides/feature.py": "override",
	}
	if !reflect.DeepEqual(got, want) || resources[len(resources)-1].target != "/main.py" {
		t.Fatalf("unexpected resource plan: %#v", resources)
	}
}
