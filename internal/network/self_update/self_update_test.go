package selfupdate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func testReleaseManifest(baseURL string) ReleaseManifest {
	m := ReleaseManifest{Schema: 1}
	m.Microsctl.Version = "0.2.0"
	m.Microsctl.URL = baseURL
	m.Microsctl.Binaries = map[string]ReleaseAsset{
		"darwin-arm64": {Path: "dist/microsctl-darwin-arm64"},
		"darwin-amd64": {Path: "dist/microsctl-darwin-amd64"},
		"linux-amd64":  {Path: "dist/microsctl-linux-amd64"},
		"linux-arm64":  {Path: "dist/microsctl-linux-arm64"},
	}
	m.MicrOS.Version = "3.6.3-0"
	return m
}

type fixtureInstaller struct {
	calls int
	body  string
}

func (i *fixtureInstaller) Install(ctx context.Context, r io.Reader) (string, error) {
	i.calls++
	data, err := io.ReadAll(r)
	i.body = string(data)
	return "backup", err
}

func TestUpdateSelectsPlatformAndOffersAnyDifferentVersion(t *testing.T) {
	for platform := range testReleaseManifest("https://example.com/").Microsctl.Binaries {
		for _, version := range []string{"0.1.0", "0.2.0", "0.3.0"} {
			t.Run(platform+"/"+version, func(t *testing.T) {
				downloaded := ""
				var data []byte
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/custom-branch/MANIFEST.yaml" {
						w.Write(data)
						return
					}
					downloaded = r.URL.Path
					io.WriteString(w, "binary")
				}))
				defer server.Close()
				manifest := testReleaseManifest(server.URL + "/custom-branch/")
				asset := ReleaseAsset{Path: "downloads/" + platform + "/custom-microsctl"}
				manifest.Microsctl.Binaries[platform] = asset
				data, _ = yaml.Marshal(manifest)
				installer := &fixtureInstaller{}
				updater := &SelfUpdater{CurrentVersion: version, Platform: platform, URL: server.URL + "/custom-branch/", Client: server.Client(), Installer: installer}
				offer, err := updater.Check(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if offer.Available != (version != "0.2.0") || offer.Platform != platform || offer.MicrOSVersion != "3.6.3-0" {
					t.Fatalf("bad offer: %+v", offer)
				}
				if installer.calls != 0 || downloaded != "" {
					t.Fatal("check installed without key press")
				}
				last := -1
				backup, err := updater.Install(context.Background(), offer, func(p int) {
					if p < last {
						t.Error("progress moved backwards")
					}
					last = p
				})
				if err != nil || backup != "backup" || installer.body != "binary" || downloaded != "/custom-branch/"+asset.Path || last != 100 {
					t.Fatalf("download=%s backup=%s err=%v", downloaded, backup, err)
				}
			})
		}
	}
}

func TestSameVersionInstallDownloadsAndReplacesExecutable(t *testing.T) {
	var manifest []byte
	downloads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/MANIFEST.yaml":
			w.Write(manifest)
		case "/dist/microsctl-linux-amd64":
			downloads++
			io.WriteString(w, "replacement payload")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	manifest, _ = yaml.Marshal(testReleaseManifest(server.URL + "/"))
	target := filepath.Join(t.TempDir(), "microsctl")
	if err := os.WriteFile(target, []byte("original payload"), 0755); err != nil {
		t.Fatal(err)
	}
	updater := &SelfUpdater{CurrentVersion: "0.2.0", Platform: "linux-amd64", URL: server.URL + "/", Client: server.Client(), Installer: ExecutableInstaller{Path: target}}
	offer, err := updater.Check(context.Background())
	if err != nil || offer.Available || downloads != 0 {
		t.Fatalf("check offer=%+v downloads=%d error=%v", offer, downloads, err)
	}
	backup, err := updater.Install(context.Background(), offer, nil)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := os.ReadFile(target)
	if err != nil || string(installed) != "replacement payload" || downloads != 1 {
		t.Fatalf("installed=%q downloads=%d error=%v", installed, downloads, err)
	}
	previous, err := os.ReadFile(backup)
	if err != nil || string(previous) != "original payload" {
		t.Fatalf("backup=%q error=%v", previous, err)
	}
}

func TestRejectsInvalidReleaseMetadata(t *testing.T) {
	for _, mutate := range []func(*ReleaseManifest){
		func(m *ReleaseManifest) { m.Schema = 2 },
		func(m *ReleaseManifest) { m.Microsctl.Version = "latest" },
		func(m *ReleaseManifest) { m.Microsctl.URL = "" },
		func(m *ReleaseManifest) { m.Microsctl.URL = "file:///tmp/revision" },
		func(m *ReleaseManifest) { m.Microsctl.URL = "https://downloads.example.com/missing-slash" },
		func(m *ReleaseManifest) { m.Microsctl.URL = "/relative/" },
	} {
		m := testReleaseManifest("https://example.com/")
		mutate(&m)
		data, _ := yaml.Marshal(m)
		if _, err := ParseReleaseManifest(data); err == nil {
			t.Fatal("accepted invalid manifest")
		}
	}
	data, _ := yaml.Marshal(testReleaseManifest("https://example.com/"))
	for _, extra := range []string{"\nunknown: true\n", "\n---\nschema: 1\n", "\nschema: 1\n"} {
		if _, err := ParseReleaseManifest(append(data, []byte(extra)...)); err == nil {
			t.Fatal("accepted extra/duplicate fields")
		}
	}
}

func TestManifestCanMoveDownloadsToAnotherHost(t *testing.T) {
	downloads := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/releases/0.2.0/custom/linux/microsctl" {
			t.Errorf("unexpected download path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, "relocated binary")
	}))
	defer downloads.Close()
	manifest := testReleaseManifest(downloads.URL + "/releases/0.2.0/")
	manifest.Microsctl.Binaries["linux-amd64"] = ReleaseAsset{Path: "custom/linux/microsctl"}
	data, _ := yaml.Marshal(manifest)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/MANIFEST.yaml" {
			t.Errorf("unexpected request to original host: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	}))
	defer server.Close()
	installer := &fixtureInstaller{}
	updater := &SelfUpdater{CurrentVersion: "0.1.0", Platform: "linux-amd64", URL: server.URL + "/", Client: server.Client(), Installer: installer}
	offer, err := updater.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// The checked offer owns its download URL, independent of later config changes.
	updater.URL = "https://example.com/another-branch/"
	if _, err := updater.Install(context.Background(), offer, nil); err != nil || installer.body != "relocated binary" {
		t.Fatalf("installed=%q error=%v", installer.body, err)
	}
}

func TestUpdateNetworkFailureAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "not found", 404) }))
	defer server.Close()
	updater := &SelfUpdater{CurrentVersion: "0.1.0", URL: server.URL + "/", Client: server.Client()}
	if _, err := updater.Check(context.Background()); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := updater.Check(ctx); err == nil {
		t.Fatal("accepted cancelled check")
	}
}

func TestUpdateRejectsWrongPlatformAndOversizedDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Length", "134217729") }))
	defer server.Close()
	installer := &fixtureInstaller{}
	updater := &SelfUpdater{CurrentVersion: "0.1.0", Platform: "linux-amd64", URL: server.URL + "/", Client: server.Client(), Installer: installer}
	offer := UpdateOffer{URL: server.URL + "/", Version: "0.2.0", Platform: "linux-arm64", Available: true, Asset: ReleaseAsset{Path: "dist/microsctl-linux-arm64"}}
	if _, err := updater.Install(context.Background(), offer, nil); err == nil {
		t.Fatal("accepted wrong platform")
	}
	offer.Platform = "linux-amd64"
	offer.Asset = testReleaseManifest("https://example.com/").Microsctl.Binaries["linux-amd64"]
	if _, err := updater.Install(context.Background(), offer, nil); err == nil || installer.calls != 0 {
		t.Fatal("accepted oversized download")
	}
}

func TestUpdateIgnoresFirmwareAndOtherPlatforms(t *testing.T) {
	m := testReleaseManifest("https://example.com/")
	m.Microsctl.Binaries = map[string]ReleaseAsset{"linux-amd64": {Path: "dist/microsctl-linux-amd64"}}
	m.MicrOS.Version = "informational"
	data, _ := yaml.Marshal(m)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer server.Close()
	updater := &SelfUpdater{CurrentVersion: "0.1.0", Platform: "linux-amd64", URL: server.URL + "/", Client: server.Client()}
	offer, err := updater.Check(context.Background())
	if err != nil || !offer.Available {
		t.Fatalf("offer=%+v err=%v", offer, err)
	}
}

func TestUpdateRejectsUnsafeAssetPathsBeforeDownload(t *testing.T) {
	var data []byte
	downloads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/MANIFEST.yaml" {
			w.Write(data)
			return
		}
		downloads++
		io.WriteString(w, "must not install")
	}))
	defer server.Close()
	installer := &fixtureInstaller{}
	updater := &SelfUpdater{CurrentVersion: "0.1.0", Platform: "linux-amd64", URL: server.URL + "/", Client: server.Client(), Installer: installer}
	for _, path := range []string{
		"", ".", "..", "../microsctl", "dist/../microsctl", "dist/./microsctl",
		"/microsctl", "//example.com/microsctl", "https://example.com/microsctl",
		"dist//microsctl", "dist/", `dist\microsctl`,
		"dist/%2e%2e/microsctl", "dist/%252e%252e/microsctl", "dist%2fmicrosctl",
		"dist/microsctl?download=1", "dist/microsctl?", "dist/microsctl#fragment",
		"dist/microsctl#", "dist/microsctl\n", "dist/%zz",
	} {
		t.Run(path, func(t *testing.T) {
			manifest := testReleaseManifest(server.URL + "/")
			asset := ReleaseAsset{Path: path}
			manifest.Microsctl.Binaries["linux-amd64"] = asset
			var err error
			data, err = yaml.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := updater.Check(context.Background()); err == nil {
				t.Fatal("check accepted unsafe asset path")
			}
			offer := UpdateOffer{URL: server.URL + "/", Version: "0.2.0", Platform: "linux-amd64", Asset: asset, Available: true}
			if _, err := updater.Install(context.Background(), offer, nil); err == nil {
				t.Fatal("install accepted unsafe asset path")
			}
			if downloads != 0 || installer.calls != 0 {
				t.Fatal("invalid path reached download or installer")
			}
		})
	}
}

func TestUpdateDoesNotFallBackToAnotherPlatform(t *testing.T) {
	manifest := testReleaseManifest("https://example.com/")
	delete(manifest.Microsctl.Binaries, "linux-amd64")
	data, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer server.Close()
	updater := &SelfUpdater{CurrentVersion: "0.1.0", Platform: "linux-amd64", URL: server.URL + "/", Client: server.Client()}
	if _, err := updater.Check(context.Background()); err == nil || !strings.Contains(err.Error(), "self-update unavailable for linux-amd64") {
		t.Fatalf("missing platform did not fail: %v", err)
	}
}
