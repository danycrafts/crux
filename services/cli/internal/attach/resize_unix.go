//go:build !windows

package attach

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/danycrafts/crux/services/cli/internal/client"
	"golang.org/x/term"
)

func startResizeMonitor(c *client.Client, sessionID string, stop chan struct{}) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	defer signal.Stop(ch)

	for {
		select {
		case <-stop:
			return
		case <-ch:
			cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
			if err == nil {
				_ = c.ResizeSession(sessionID, uint16(rows), uint16(cols))
			}
		}
	}
}
