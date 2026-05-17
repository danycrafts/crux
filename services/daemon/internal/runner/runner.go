package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/danycrafts/crux/services/daemon/internal/store"
)

// SessionRunner manages a single agent session.
type SessionRunner struct {
	store    *store.Store
	mu       sync.RWMutex
	handles  map[string]*PTYHandle
	readers  map[string]*broadcastReader
}

// NewSessionRunner creates a runner backed by the store.
func NewSessionRunner(s *store.Store) *SessionRunner {
	return &SessionRunner{
		store:   s,
		handles: make(map[string]*PTYHandle),
		readers: make(map[string]*broadcastReader),
	}
}

// PTYHandle abstracts a PTY or pipe session.
type PTYHandle struct {
	SessionID string
	Stdin     io.Writer
	Stdout    io.Reader
	Close     func() error
	Wait      func() error
	resize    func(rows, cols uint16) error
	mu        sync.Mutex
	closed    bool
}

// Resize updates the terminal size if supported.
func (h *PTYHandle) Resize(rows, cols uint16) error {
	if h.resize != nil {
		return h.resize(rows, cols)
	}
	return nil
}

// PTYFactory creates platform-specific PTY handles.
type PTYFactory interface {
	Start(cmd *exec.Cmd) (*PTYHandle, error)
}

var factory PTYFactory

func init() {
	factory = newPlatformFactory()
}

// StartSession spawns an agent session and records it.
func (r *SessionRunner) StartSession(ctx context.Context, agentID, sessionID, workDir string, env []string) (*PTYHandle, error) {
	cmdPath, extraArgs, err := resolveAgent(agentID)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, cmdPath, extraArgs...)
	cmd.Dir = workDir
	if len(env) > 0 {
		cmd.Env = env
	} else {
		cmd.Env = os.Environ()
	}

	handle, err := factory.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("pty start: %w", err)
	}
	handle.SessionID = sessionID

	// Record session start
	sess := &store.Session{
		ID:        sessionID,
		AgentID:   agentID,
		RepoPath:  workDir,
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	_ = r.store.CreateSession(ctx, sess)

	// Register handle
	r.mu.Lock()
	r.handles[sessionID] = handle
	br := newBroadcastReader(handle.Stdout)
	r.readers[sessionID] = br
	r.mu.Unlock()

	// Start broadcast
	go br.run()

	// Background transcript capture
	go r.capture(sessionID, br)

	// Background exit detection
	go func() {
		_ = handle.Wait()
		end := time.Now().UTC()
		_ = r.store.UpdateSession(context.Background(), &store.Session{
			ID:      sessionID,
			Status:  "exited",
			EndedAt: &end,
		})
		r.mu.Lock()
		delete(r.handles, sessionID)
		delete(r.readers, sessionID)
		r.mu.Unlock()
	}()

	return handle, nil
}

// GetHandle returns the active handle for a session.
func (r *SessionRunner) GetHandle(sessionID string) (*PTYHandle, *broadcastReader, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handles[sessionID]
	br, _ := r.readers[sessionID]
	return h, br, ok
}

// SendInput writes user input to the session and records it.
func (r *SessionRunner) SendInput(ctx context.Context, sessionID string, handle *PTYHandle, data []byte) error {
	if handle == nil || handle.Stdin == nil {
		return fmt.Errorf("session %s not active", sessionID)
	}
	_, err := handle.Stdin.Write(data)
	if err != nil {
		return err
	}
	// Best-effort transcript
	_ = r.store.AppendTranscript(ctx, sessionID, string(data), true)
	return nil
}

// StopSession kills the session process.
func (r *SessionRunner) StopSession(sessionID string) error {
	r.mu.Lock()
	handle, ok := r.handles[sessionID]
	if ok {
		delete(r.handles, sessionID)
		delete(r.readers, sessionID)
	}
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return handle.Close()
}

func (r *SessionRunner) capture(sessionID string, br *broadcastReader) {
	for data := range br.C {
		_ = r.store.AppendTranscript(context.Background(), sessionID, string(data), false)
	}
}

// broadcastReader fans out PTY stdout to multiple consumers.
type broadcastReader struct {
	src io.Reader
	C   chan []byte
	mu  sync.RWMutex
	subs []chan []byte
}

func newBroadcastReader(src io.Reader) *broadcastReader {
	return &broadcastReader{
		src: src,
		C:   make(chan []byte, 100),
		subs: make([]chan []byte, 0),
	}
}

func (b *broadcastReader) run() {
	buf := make([]byte, 4096)
	for {
		n, err := b.src.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case b.C <- data:
			default:
			}
			b.mu.RLock()
			subs := b.subs
			b.mu.RUnlock()
			for _, ch := range subs {
				select {
				case ch <- data:
				default:
				}
			}
		}
		if err != nil {
			close(b.C)
			b.mu.Lock()
			for _, ch := range b.subs {
				close(ch)
			}
			b.subs = nil
			b.mu.Unlock()
			return
		}
	}
}

// Subscribe returns a new channel that receives stdout data.
func (b *broadcastReader) Subscribe() chan []byte {
	ch := make(chan []byte, 100)
	b.mu.Lock()
	b.subs = append(b.subs, ch)
	b.mu.Unlock()
	return ch
}

// Registry of agent commands.
var agentRegistry sync.RWMutex
var agentCommands = map[string]string{
	"claude-code": "claude",
	"codex":       "codex",
	"gemini-cli":  "gemini",
	"opencode":    "opencode",
	"aider":       "aider",
}

// RegisterAgentCommand allows runtime agent registration.
func RegisterAgentCommand(id, cmd string) {
	agentRegistry.Lock()
	defer agentRegistry.Unlock()
	agentCommands[id] = cmd
}

func resolveAgent(agentID string) (string, []string, error) {
	agentRegistry.RLock()
	cmdName, ok := agentCommands[agentID]
	agentRegistry.RUnlock()
	if !ok {
		return "", nil, fmt.Errorf("unknown agent %q", agentID)
	}
	path, err := exec.LookPath(cmdName)
	if err != nil {
		return "", nil, fmt.Errorf("agent command %q not found: %w", cmdName, err)
	}
	return path, nil, nil
}
