package widgets

import (
	"github.com/micros/microsctl/internal/usb"
	"strings"
)

func (m State) BoardSelectorWidget() string {
	var buttons []string
	for _, board := range usb.Boards(m.Inventory.Images) {
		label := "[" + Clean(board) + "]"
		if board == m.BoardType {
			label = StyleKey.Render(label)
		} else {
			label = StyleMuted.Render(label)
		}
		buttons = append(buttons, label)
	}
	return strings.Join(buttons, " ") + "  " + StyleMuted.Render("←/→ board")
}
