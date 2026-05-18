package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
}

// New opens or creates the SQLite store.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Improve concurrency with WAL and busy timeout
	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	_, _ = db.Exec("PRAGMA busy_timeout=5000")
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS agents (
	id TEXT PRIMARY KEY,
	name TEXT,
	agent_type TEXT,
	provider TEXT,
	command TEXT,
	version TEXT,
	owner TEXT,
	capabilities TEXT,
	status TEXT,
	created_at TEXT,
	updated_at TEXT
);

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	agent_id TEXT,
	project TEXT,
	repo_path TEXT,
	status TEXT,
	started_at TEXT,
	ended_at TEXT,
	cost_usd REAL,
	tool_calls INTEGER,
	fallbacks INTEGER,
	summary TEXT,
	continuation TEXT
);

CREATE TABLE IF NOT EXISTS events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT,
	event_type TEXT,
	timestamp TEXT,
	payload TEXT,
	FOREIGN KEY(session_id) REFERENCES sessions(id)
);

CREATE TABLE IF NOT EXISTS transcripts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT,
	line TEXT,
	is_input BOOLEAN,
	created_at TEXT,
	FOREIGN KEY(session_id) REFERENCES sessions(id)
);

CREATE TABLE IF NOT EXISTS mcp_calls (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT,
	tool_name TEXT,
	server_name TEXT,
	status TEXT,
	latency_ms INTEGER,
	cost_usd REAL,
	created_at TEXT,
	FOREIGN KEY(session_id) REFERENCES sessions(id)
);

CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id);
CREATE INDEX IF NOT EXISTS idx_transcripts_session ON transcripts(session_id);
CREATE INDEX IF NOT EXISTS idx_mcp_calls_session ON mcp_calls(session_id);
`
	_, err := s.db.Exec(schema)
	return err
}

// Agent represents a registered agent.
type Agent struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Provider     string    `json:"provider"`
	Command      string    `json:"command"`
	Version      string    `json:"version"`
	Owner        string    `json:"owner"`
	Capabilities []string  `json:"capabilities"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UpsertAgent inserts or updates an agent.
func (s *Store) UpsertAgent(ctx context.Context, a *Agent) error {
	caps, _ := json.Marshal(a.Capabilities)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (id, name, agent_type, provider, command, version, owner, capabilities, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, agent_type=excluded.agent_type, provider=excluded.provider,
			command=excluded.command, version=excluded.version, owner=excluded.owner,
			capabilities=excluded.capabilities, status=excluded.status, updated_at=excluded.updated_at
	`, a.ID, a.Name, a.Type, a.Provider, a.Command, a.Version, a.Owner, string(caps), a.Status, now, now)
	return err
}

// ListAgents returns all agents.
func (s *Store) ListAgents(ctx context.Context) ([]Agent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, agent_type, provider, command, version, owner, capabilities, status, created_at, updated_at FROM agents`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Agent
	for rows.Next() {
		var a Agent
		var caps, created, updated string
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Provider, &a.Command, &a.Version, &a.Owner, &caps, &a.Status, &created, &updated); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(caps), &a.Capabilities)
		a.CreatedAt, _ = time.Parse(time.RFC3339, created)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, a)
	}
	return out, rows.Err()
}

// Session represents an agent session.
type Session struct {
	ID           string     `json:"id"`
	AgentID      string     `json:"agent_id"`
	Project      string     `json:"project"`
	RepoPath     string     `json:"repo_path"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	CostUSD      float64    `json:"cost_usd"`
	ToolCalls    int        `json:"tool_calls"`
	Fallbacks    int        `json:"fallbacks"`
	Summary      string     `json:"summary"`
	Continuation string     `json:"continuation"`
}

// CreateSession starts a new session.
func (s *Store) CreateSession(ctx context.Context, sess *Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, agent_id, project, repo_path, status, started_at, cost_usd, tool_calls, fallbacks, summary, continuation)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sess.ID, sess.AgentID, sess.Project, sess.RepoPath, sess.Status, sess.StartedAt.Format(time.RFC3339), sess.CostUSD, sess.ToolCalls, sess.Fallbacks, sess.Summary, sess.Continuation)
	return err
}

// UpdateSession updates mutable session fields.
func (s *Store) UpdateSession(ctx context.Context, sess *Session) error {
	var ended *string
	if sess.EndedAt != nil {
		v := sess.EndedAt.Format(time.RFC3339)
		ended = &v
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions SET agent_id=?, project=?, repo_path=?, status=?, ended_at=?, cost_usd=?, tool_calls=?, fallbacks=?, summary=?, continuation=?
		WHERE id=?
	`, sess.AgentID, sess.Project, sess.RepoPath, sess.Status, ended, sess.CostUSD, sess.ToolCalls, sess.Fallbacks, sess.Summary, sess.Continuation, sess.ID)
	return err
}

// GetSession returns a single session.
func (s *Store) GetSession(ctx context.Context, id string) (*Session, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, agent_id, project, repo_path, status, started_at, ended_at, cost_usd, tool_calls, fallbacks, summary, continuation FROM sessions WHERE id=?`, id)
	var sess Session
	var started string
	var ended sql.NullString
	if err := row.Scan(&sess.ID, &sess.AgentID, &sess.Project, &sess.RepoPath, &sess.Status, &started, &ended, &sess.CostUSD, &sess.ToolCalls, &sess.Fallbacks, &sess.Summary, &sess.Continuation); err != nil {
		return nil, err
	}
	sess.StartedAt, _ = time.Parse(time.RFC3339, started)
	if ended.Valid && ended.String != "" {
		t, _ := time.Parse(time.RFC3339, ended.String)
		sess.EndedAt = &t
	}
	return &sess, nil
}

// ListSessions returns sessions ordered by start time desc.
func (s *Store) ListSessions(ctx context.Context, limit int) ([]Session, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, agent_id, project, repo_path, status, started_at, ended_at, cost_usd, tool_calls, fallbacks, summary, continuation FROM sessions ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Session
	for rows.Next() {
		var sess Session
		var started string
		var ended sql.NullString
		if err := rows.Scan(&sess.ID, &sess.AgentID, &sess.Project, &sess.RepoPath, &sess.Status, &started, &ended, &sess.CostUSD, &sess.ToolCalls, &sess.Fallbacks, &sess.Summary, &sess.Continuation); err != nil {
			return nil, err
		}
		sess.StartedAt, _ = time.Parse(time.RFC3339, started)
		if ended.Valid && ended.String != "" {
			t, _ := time.Parse(time.RFC3339, ended.String)
			sess.EndedAt = &t
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// Event is a normalized session event.
type Event struct {
	ID        int64           `json:"id"`
	SessionID string          `json:"session_id"`
	Type      string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// InsertEvent records an event.
func (s *Store) InsertEvent(ctx context.Context, e *Event) error {
	res, err := s.db.ExecContext(ctx, `INSERT INTO events (session_id, event_type, timestamp, payload) VALUES (?, ?, ?, ?)`,
		e.SessionID, e.Type, e.Timestamp.Format(time.RFC3339), string(e.Payload))
	if err != nil {
		return err
	}
	e.ID, _ = res.LastInsertId()
	return nil
}

// ListEvents returns events for a session.
func (s *Store) ListEvents(ctx context.Context, sessionID string) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, session_id, event_type, timestamp, payload FROM events WHERE session_id=? ORDER BY id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		var ts string
		var payload string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Type, &ts, &payload); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, ts)
		e.Payload = json.RawMessage(payload)
		out = append(out, e)
	}
	return out, rows.Err()
}

// AppendTranscript adds a transcript line.
func (s *Store) AppendTranscript(ctx context.Context, sessionID string, line string, isInput bool) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO transcripts (session_id, line, is_input, created_at) VALUES (?, ?, ?, ?)`,
		sessionID, line, isInput, time.Now().UTC().Format(time.RFC3339))
	return err
}

// GetTranscript returns transcript lines for a session.
func (s *Store) GetTranscript(ctx context.Context, sessionID string) ([]TranscriptLine, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT line, is_input, created_at FROM transcripts WHERE session_id=? ORDER BY id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TranscriptLine
	for rows.Next() {
		var t TranscriptLine
		var ts string
		if err := rows.Scan(&t.Line, &t.IsInput, &ts); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, ts)
		out = append(out, t)
	}
	return out, rows.Err()
}

// TranscriptLine is a single transcript entry.
type TranscriptLine struct {
	Line      string    `json:"line"`
	IsInput   bool      `json:"is_input"`
	CreatedAt time.Time `json:"created_at"`
}

// Stats returns aggregate stats.
func (s *Store) Stats(ctx context.Context) (map[string]interface{}, error) {
	var totalSessions, activeSessions, totalToolCalls int
	var totalCost float64
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(tool_calls),0), COALESCE(SUM(cost_usd),0) FROM sessions`)
	if err := row.Scan(&totalSessions, &totalToolCalls, &totalCost); err != nil {
		return nil, err
	}
	row = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE status='running'`)
	_ = row.Scan(&activeSessions)

	return map[string]interface{}{
		"total_sessions":   totalSessions,
		"active_sessions":  activeSessions,
		"total_tool_calls": totalToolCalls,
		"total_cost_usd":   fmt.Sprintf("%.2f", totalCost),
	}, nil
}

// MCPCall represents a single MCP tool invocation.
type MCPCall struct {
	ID         int64     `json:"id"`
	SessionID  string    `json:"session_id"`
	ToolName   string    `json:"tool_name"`
	ServerName string    `json:"server_name"`
	Status     string    `json:"status"`
	LatencyMs  int64     `json:"latency_ms"`
	CostUSD    float64   `json:"cost_usd"`
	CreatedAt  time.Time `json:"created_at"`
}

// InsertMCPCall records an MCP call.
func (s *Store) InsertMCPCall(ctx context.Context, c *MCPCall) error {
	res, err := s.db.ExecContext(ctx, `INSERT INTO mcp_calls (session_id, tool_name, server_name, status, latency_ms, cost_usd, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.SessionID, c.ToolName, c.ServerName, c.Status, c.LatencyMs, c.CostUSD, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

// ListMCPCalls returns MCP calls for a session or all if sessionID is empty.
func (s *Store) ListMCPCalls(ctx context.Context, sessionID string) ([]MCPCall, error) {
	var rows *sql.Rows
	var err error
	if sessionID != "" {
		rows, err = s.db.QueryContext(ctx, `SELECT id, session_id, tool_name, server_name, status, latency_ms, cost_usd, created_at FROM mcp_calls WHERE session_id=? ORDER BY id DESC`, sessionID)
	} else {
		rows, err = s.db.QueryContext(ctx, `SELECT id, session_id, tool_name, server_name, status, latency_ms, cost_usd, created_at FROM mcp_calls ORDER BY id DESC LIMIT 100`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MCPCall
	for rows.Next() {
		var c MCPCall
		var ts string
		if err := rows.Scan(&c.ID, &c.SessionID, &c.ToolName, &c.ServerName, &c.Status, &c.LatencyMs, &c.CostUSD, &ts); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, ts)
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateSessionSummary updates just the summary field.
func (s *Store) UpdateSessionSummary(ctx context.Context, sessionID, summary string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET summary=? WHERE id=?`, summary, sessionID)
	return err
}
