package network

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func defaultWebUIURL(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return ""
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return (&url.URL{Scheme: "http", Host: host}).String()
}

// Probe only the default HTTP endpoint, without credentials or redirects.
func webUIAvailable(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	transport := &http.Transport{DialContext: (&net.Dialer{Timeout: time.Second}).DialContext}
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: 2 * time.Second, Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK || response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden
}

// WebUIURL uses the reachable address for special endpoints and mDNS for LAN nodes.
func (d Device) WebUIURL() string {
	if !strings.EqualFold(d.Features["webui"], "ON") {
		return ""
	}
	if d.Online && d.ProbedWebUI != "" {
		return d.ProbedWebUI
	}
	switch d.SpecialEndpoint() {
	case "Localhost":
		return "http://localhost"
	case "AP mode":
		return "http://192.168.4.1"
	}
	if len(d.Name) == 0 || len(d.Name) > 63 {
		return ""
	}
	for i, c := range d.Name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}
		if c != '-' || i == 0 || i == len(d.Name)-1 {
			return ""
		}
	}
	return "http://" + d.Name + ".local"
}
