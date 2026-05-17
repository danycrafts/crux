//go:build windows

package runner

import (
	"io"
	"os/exec"
	"sync"
)

type windowsFactory struct{}

func newPlatformFactory() PTYFactory {
	return &windowsFactory{}
}

func (f *windowsFactory) Start(cmd *exec.Cmd) (*PTYHandle, error) {
	// Windows pseudo-consoles are complex; for MVP use stdin/stdout pipes.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var once sync.Once
	closeFn := func() error {
		once.Do(func() {
			stdin.Close()
			cmd.Process.Kill()
		})
		return nil
	}
	return &PTYHandle{
		Stdin:  stdin,
		Stdout: io.MultiReader(stdout, stderr),
		Close:  closeFn,
		Wait:   cmd.Wait,
		resize: nil,
	}, nil
}
