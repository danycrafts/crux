//go:build !windows

package runner

import (
	"os/exec"

	"github.com/creack/pty"
)

type unixFactory struct{}

func newPlatformFactory() PTYFactory {
	return &unixFactory{}
}

func (f *unixFactory) Start(cmd *exec.Cmd) (*PTYHandle, error) {
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	return &PTYHandle{
		Stdin:  ptmx,
		Stdout: ptmx,
		Close:  func() error { return ptmx.Close() },
		Wait:   cmd.Wait,
		resize: func(rows, cols uint16) error {
			return pty.Setsize(ptmx, &pty.Winsize{Rows: rows, Cols: cols})
		},
	}, nil
}
