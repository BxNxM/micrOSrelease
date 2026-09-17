package usb

import "sort"

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

func BoardImages(images []Image, board string) []int {
	var indices []int
	for i, image := range images {
		if board == "" || image.Board == board {
			indices = append(indices, i)
		}
	}
	return indices
}
