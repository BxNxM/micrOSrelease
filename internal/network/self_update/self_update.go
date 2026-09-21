package selfupdate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"
)

type UpdateOffer struct {
	Version, MicrOSVersion, Platform string
	URL                              string
	Asset                            ReleaseAsset
	Available                        bool
}

type BinaryInstaller interface {
	Install(context.Context, io.Reader) (string, error)
}

type AppUpdater interface {
	Check(context.Context) (UpdateOffer, error)
	Install(context.Context, UpdateOffer, func(int)) (string, error)
}

type SelfUpdater struct {
	CurrentVersion string
	Installer      BinaryInstaller
	Client         *http.Client
	URL            string
	// Platform is injectable for local fixture tests.
	Platform string
}

func (s *SelfUpdater) client() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 || req.URL.Scheme != "https" || req.URL.Host != via[0].URL.Host {
			return fmt.Errorf("unexpected release redirect")
		}
		return nil
	}}
}
func (s *SelfUpdater) platform() string {
	if s.Platform != "" {
		return s.Platform
	}
	return runtime.GOOS + "-" + runtime.GOARCH
}

func (s *SelfUpdater) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "microsctl/"+s.CurrentVersion)
	response, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("release server returned HTTP %d", response.StatusCode)
	}
	return response, nil
}

func (s *SelfUpdater) Check(ctx context.Context) (UpdateOffer, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := validateReleaseURL(s.URL); err != nil {
		return UpdateOffer{}, err
	}
	response, err := s.get(ctx, s.URL+"MANIFEST.yaml")
	if err != nil {
		return UpdateOffer{}, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return UpdateOffer{}, err
	}
	if len(data) > 1<<20 {
		return UpdateOffer{}, fmt.Errorf("release manifest exceeds 1 MiB")
	}
	manifest, err := ParseReleaseManifest(data)
	if err != nil {
		return UpdateOffer{}, err
	}
	comparison, err := CompareReleaseVersions(manifest.Microsctl.Version, s.CurrentVersion)
	if err != nil {
		return UpdateOffer{}, err
	}
	platform := s.platform()
	asset, ok := manifest.Microsctl.Binaries[platform]
	if !ok || asset.Path != "dist/"+BinaryName(platform) {
		return UpdateOffer{}, fmt.Errorf("self-update unavailable for %s", platform)
	}
	return UpdateOffer{Version: manifest.Microsctl.Version, MicrOSVersion: manifest.MicrOS.Version, Platform: platform, URL: manifest.Microsctl.URL, Asset: asset, Available: comparison != 0}, nil
}

func (s *SelfUpdater) Install(ctx context.Context, offer UpdateOffer, emit func(int)) (string, error) {
	// Availability controls automatic offers; an explicit install may reinstall
	// the current version using the same validated platform and download path.
	_, err := CompareReleaseVersions(offer.Version, s.CurrentVersion)
	if err != nil || offer.Platform != s.platform() || offer.Asset.Path != "dist/"+BinaryName(s.platform()) {
		return "", fmt.Errorf("invalid update offer")
	}
	if s.Installer == nil {
		return "", fmt.Errorf("binary installer unavailable")
	}
	if err := validateReleaseURL(offer.URL); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	response, err := s.get(ctx, offer.URL+offer.Asset.Path)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.ContentLength > MaxReleaseSize {
		return "", fmt.Errorf("download exceeds size limit")
	}
	progress := &updateProgress{reader: response.Body, total: response.ContentLength, emit: emit, last: -1}
	backup, err := s.Installer.Install(ctx, progress)
	if err != nil {
		return backup, err
	}
	if emit != nil {
		emit(100)
	}
	return backup, nil
}

type updateProgress struct {
	reader      io.Reader
	read, total int64
	last        int
	emit        func(int)
}

func (p *updateProgress) Read(buffer []byte) (int, error) {
	n, err := p.reader.Read(buffer)
	p.read += int64(n)
	if p.total > 0 {
		percentage := min(99, int(p.read*100/p.total))
		if percentage != p.last && p.emit != nil {
			p.emit(percentage)
			p.last = percentage
		}
	}
	return n, err
}
