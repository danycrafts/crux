//go:build windows

package attach

import (
	"os"
	"time"

	"github.com/danycrafts/crux/services/cli/internal/client"
	"golang.org/x/term"
)

func startResizeMonitor(c *client.Client, sessionID string, stop chan struct{}) {
	// Windows: poll terminal size every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastCols, lastRows int
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
			if err == nil && (cols != lastCols || rows != lastRows) {
				lastCols, lastRows = cols, rows
				_ = c.ResizeSession(sessionID, uint16(rows), uint16(cols))
			}
		}
	}
}
