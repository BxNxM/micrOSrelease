package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	tea "charm.land/bubbletea/v2"
)

type browserMsg struct{ err error }

func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.CommandContext(ctx, "open", url)
		case "windows":
			cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url)
		case "linux":
			cmd = exec.CommandContext(ctx, "xdg-open", url)
		default:
			return browserMsg{err: fmt.Errorf("unsupported platform: %s", runtime.GOOS)}
		}
		return browserMsg{err: cmd.Run()}
	}
}
