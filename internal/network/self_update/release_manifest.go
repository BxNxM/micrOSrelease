package selfupdate

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const MaxReleaseSize int64 = 128 << 20

var releaseVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

type ReleaseAsset struct {
	Path string `yaml:"path"`
}

func validateReleaseURL(value string) error {
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !strings.HasSuffix(value, "/") {
		return fmt.Errorf("invalid release URL %q", value)
	}
	return nil
}

type ReleaseManifest struct {
	Schema    int `yaml:"schema"`
	Microsctl struct {
		Version  string                  `yaml:"version"`
		URL      string                  `yaml:"url"`
		Binaries map[string]ReleaseAsset `yaml:"binaries"`
	} `yaml:"microsctl"`
	MicrOS struct {
		Version string `yaml:"version"`
	} `yaml:"micros"`
}

// Asset paths are unescaped relative URL paths beneath the release URL.
func validateReleasePath(value string) error {
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || u.Host != "" || u.Path != value ||
		!fs.ValidPath(value) || value == "." || strings.Contains(value, `\`) {
		return fmt.Errorf("invalid release asset path %q", value)
	}
	return nil
}

func ParseReleaseManifest(data []byte) (ReleaseManifest, error) {
	var manifest ReleaseManifest
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("invalid release manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return manifest, fmt.Errorf("release manifest must contain one document")
	}
	return manifest, manifest.Validate()
}

func (m ReleaseManifest) Validate() error {
	if m.Schema != 1 {
		return fmt.Errorf("unsupported release manifest schema %d", m.Schema)
	}
	if _, err := CompareReleaseVersions(m.Microsctl.Version, m.Microsctl.Version); err != nil {
		return err
	}

	return validateReleaseURL(m.Microsctl.URL)
}

// CompareReleaseVersions compares stable numeric versions.
func CompareReleaseVersions(left, right string) (int, error) {
	if !releaseVersion.MatchString(left) || !releaseVersion.MatchString(right) {
		return 0, fmt.Errorf("microsctl versions must use major.minor.patch")
	}
	l, r := strings.Split(left, "."), strings.Split(right, ".")
	for i := range l {
		a, err := strconv.ParseUint(l[i], 10, 32)
		if err != nil {
			return 0, err
		}
		b, err := strconv.ParseUint(r[i], 10, 32)
		if err != nil {
			return 0, err
		}
		if a > b {
			return 1, nil
		}
		if a < b {
			return -1, nil
		}
	}
	return 0, nil
}
