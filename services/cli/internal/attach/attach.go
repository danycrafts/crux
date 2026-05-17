package attach

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/danycrafts/crux/services/cli/internal/client"
	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

// Session attaches the local terminal to a remote PTY session via WebSocket.
func Session(c *client.Client, sessionID string) error {
	ws, err := c.AttachSession(sessionID)
	if err != nil {
		return fmt.Errorf("attach websocket: %w", err)
	}
	defer ws.Close()

	// Save terminal state and enter raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("make raw: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Initial resize
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil {
		_ = c.ResizeSession(sessionID, uint16(rows), uint16(cols))
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// stdin -> websocket
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// websocket -> stdout
	go func() {
		defer wg.Done()
		for {
			mt, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.BinaryMessage || mt == websocket.TextMessage {
				_, _ = os.Stdout.Write(data)
			}
		}
	}()

	// Handle terminal resize
	stopResize := make(chan struct{})
	go monitorResize(c, sessionID, stopResize)

	wg.Wait()
	close(stopResize)
	return nil
}

// SessionNonInteractive attaches without raw mode for non-TTY use.
func SessionNonInteractive(c *client.Client, sessionID string) error {
	ws, err := c.AttachSession(sessionID)
	if err != nil {
		return fmt.Errorf("attach websocket: %w", err)
	}
	defer ws.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			mt, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.BinaryMessage || mt == websocket.TextMessage {
				_, _ = os.Stdout.Write(data)
			}
		}
	}()

	wg.Wait()
	return nil
}

func monitorResize(c *client.Client, sessionID string, stop chan struct{}) {
	// Platform-specific resize monitoring
	startResizeMonitor(c, sessionID, stop)
}

// CopyStreams is a simple bidirectional copy for HTTP fallback.
func CopyStreams(input io.Writer, output io.Reader) error {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(input, os.Stdin)
	}()
	go func() {
		defer wg.Done()
		io.Copy(os.Stdout, output)
	}()
	wg.Wait()
	return nil
}
