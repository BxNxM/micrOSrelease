package network

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func inspect(ctx context.Context, node Device, password string) (Device, bool) {
	node.CheckedAt = time.Now()
	node.Cached = false
	node.Online = false
	node.Version, node.Mode = "n/a", "n/a"
	node.Features = map[string]string{}
	node.Latency = 0
	node.Error = ""
	client, err := Dial(ctx, node.Address, password)
	if err != nil {
		node.Error = err.Error()
		return node, false
	}
	defer client.Close()
	start := time.Now()
	hello, err := client.Command(ctx, "hello")
	if err != nil {
		node.Error = err.Error()
		return node, false
	}
	var fields []string
	for _, line := range strings.Split(hello, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "hello:") {
			fields = strings.Split(strings.TrimSpace(line), ":")
			break
		}
	}
	if len(fields) < 3 || fields[1] == "" || fields[2] == "" {
		node.Error = "not a micrOS hello response"
		return node, false
	}
	node.Name, node.UID, node.Online = fields[1], fields[2], true
	if len(fields) > 3 {
		node.Mode = fields[3]
	}
	node.Version, err = client.Command(ctx, "version")
	node.Latency = time.Since(start)
	if err != nil {
		node.Version = "n/a"
		node.Error = err.Error()
		return node, true
	}
	if _, err = client.Command(ctx, "conf"); err != nil {
		node.Error = err.Error()
		return node, true
	}
	for _, key := range []string{"webui", "espnow", "auth", "cron", "timirq"} {
		value, err := client.Command(ctx, key)
		if err != nil {
			node.Error = fmt.Sprintf("%s: %v", key, err)
			break
		}
		switch strings.TrimSpace(value) {
		case "True":
			node.Features[key] = "ON"
		case "False":
			node.Features[key] = "OFF"
		}
	}
	return node, true
}
