package usb

import (
	"regexp"
	"strings"
)

var runtimeChipPattern = regexp.MustCompile(`(?i)\besp(?:8266|32)(?:[-_ ]?[a-z][0-9]+)?(?:[-_ ]?rev[0-9]+)?\b`)

func runtimeChip(machine string) string {
	// uname commonly reads "<board name> with <chip>". Use the final chip,
	// so an ESP32-branded board name cannot hide a more specific C/S variant.
	matches := runtimeChipPattern.FindAllString(machine, -1)
	if len(matches) == 0 {
		return ""
	}
	return normalizeChip(matches[len(matches)-1])
}

func supportedESPChip(chip string) bool {
	switch normalizeChip(chip) {
	case "esp8266", "esp32", "esp32s2", "esp32s3", "esp32c2", "esp32c3", "esp32c5", "esp32c6", "esp32h2", "esp32p4rev1":
		return true
	default:
		return false
	}
}

func normalizeChip(chip string) string {
	replacer := strings.NewReplacer("-", "", "_", "", " ", "")
	return replacer.Replace(strings.ToLower(chip))
}
