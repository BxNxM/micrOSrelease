package usb

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func Boards(images []Image) []string {
	seen := make(map[string]bool)
	var boards []string
	for _, image := range images {
		if !seen[image.Board] {
			seen[image.Board] = true
			boards = append(boards, image.Board)
		}
	}
	sort.Strings(boards)
	return boards
}

// BoardImages returns inventory indices ordered by newest micrOS version first,
// using the MicroPython version to break ties without reordering the inventory.
func BoardImages(images []Image, board string) []int {
	var indices []int
	for i, image := range images {
		if board == "" || image.Board == board {
			indices = append(indices, i)
		}
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left, right := images[indices[i]], images[indices[j]]
		if order := compareVersion(left.Version, right.Version); order != 0 {
			return order > 0
		}
		return compareVersion(left.MicroPython, right.MicroPython) > 0
	})
	return indices
}

// LatestBoardImage returns the newest micrOS release for a board. Versions use
// numeric comparison so, for example, 3.10.0 sorts after 3.9.0.
func LatestBoardImage(images []Image, board string) (int, bool) {
	latest := -1
	for index, image := range images {
		if image.Board != board {
			continue
		}
		if latest < 0 || compareVersion(image.Version, images[latest].Version) > 0 ||
			(compareVersion(image.Version, images[latest].Version) == 0 && compareVersion(image.MicroPython, images[latest].MicroPython) > 0) {
			latest = index
		}
	}
	return latest, latest >= 0
}

func compareVersion(left, right string) int {
	leftParts := numericVersionParts(left)
	rightParts := numericVersionParts(right)
	for index := 0; index < max(len(leftParts), len(rightParts)); index++ {
		var leftPart, rightPart int
		if index < len(leftParts) {
			leftPart = leftParts[index]
		}
		if index < len(rightParts) {
			rightPart = rightParts[index]
		}
		if leftPart < rightPart {
			return -1
		}
		if leftPart > rightPart {
			return 1
		}
	}
	return strings.Compare(left, right)
}

func numericVersionParts(version string) []int {
	fields := strings.FieldsFunc(version, func(character rune) bool { return !unicode.IsDigit(character) })
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		part, err := strconv.Atoi(field)
		if err == nil {
			parts = append(parts, part)
		}
	}
	return parts
}

// BoardForChip finds a bundled board using the same chip names as USB validation.
func BoardForChip(images []Image, chip string) (string, bool) {
	for _, board := range Boards(images) {
		if normalizeChip(board) == normalizeChip(chip) {
			return board, true
		}
	}
	return "", false
}
