package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/danycrafts/crux/pkg/logger"
	"github.com/danycrafts/crux/services/daemon/internal/config"
	"github.com/danycrafts/crux/services/daemon/internal/discovery"
	"github.com/danycrafts/crux/services/daemon/internal/gateway"
	"github.com/danycrafts/crux/services/daemon/internal/runner"
	"github.com/danycrafts/crux/services/daemon/internal/store"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:      func(r *http.Request) bool { return true },
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
}

// Server is the HTTP API for the daemon.
type Server struct {
	cfg    *config.Config
	store  *store.Store
	runner *runner.SessionRunner
	mux    *http.ServeMux
	srv    *http.Server
}

// NewServer creates the API server.
func NewServer(cfg *config.Config, st *store.Store) *Server {
	s := &Server{
		cfg:    cfg,
		store:  st,
		runner: runner.NewSessionRunner(st),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /discover", s.handleDiscover)
	mux.HandleFunc("GET /agents", s.handleListAgents)
	mux.HandleFunc("POST /agents/{id}/run", s.handleRunAgent)
	mux.HandleFunc("POST /sessions/{id}/input", s.handleSessionInput)
	mux.HandleFunc("GET /sessions/{id}/attach", s.handleSessionAttach)
	mux.HandleFunc("POST /sessions/{id}/resize", s.handleSessionResize)
	mux.HandleFunc("POST /sessions/{id}/stop", s.handleSessionStop)
	mux.HandleFunc("GET /sessions", s.handleListSessions)
	mux.HandleFunc("GET /sessions/{id}", s.handleGetSession)
	mux.HandleFunc("GET /sessions/{id}/logs", s.handleSessionLogs)
	mux.HandleFunc("GET /sessions/{id}/events", s.handleSessionEvents)
	mux.HandleFunc("GET /sessions/{id}/summary", s.handleSessionSummary)
	mux.HandleFunc("POST /sessions/{id}/replay", s.handleSessionReplay)
	mux.HandleFunc("POST /sessions/{id}/continue", s.handleContinueSession)
	mux.HandleFunc("GET /mcp/servers", s.handleMCPList)
	mux.HandleFunc("GET /mcp/tools", s.handleMCPTools)
	mux.HandleFunc("GET /mcp/calls", s.handleMCPCalls)
	mux.HandleFunc("POST /mcp/generate", s.handleMCPGenerate)
	mux.HandleFunc("GET /mcp/policy", s.handleMCPPolicy)
	mux.HandleFunc("POST /mcp/policy", s.handleMCPPolicyUpdate)
	mux.HandleFunc("GET /stats", s.handleStats)
	s.mux = mux
	return s
}

// Start begins listening.
func (s *Server) Start(addr string) error {
	s.srv = &http.Server{Addr: addr, Handler: s.mux}
	return s.srv.ListenAndServe()
}

// Stop shuts down gracefully.
func (s *Server) Stop(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	logger.Info("discovering agents")
	found := discovery.Discover(r.Context())
	for _, a := range found {
		_ = s.store.UpsertAgent(r.Context(), &store.Agent{
			ID:           slug(a.Name),
			Name:         a.Name,
			Type:         "cli",
			Provider:     a.Provider,
			Command:      a.Command,
			Capabilities: a.Capabilities,
			Status:       "available",
		})
	}

	// Deep discovery: parse agent local data
	agentData := discovery.DiscoverAllAgentData()
	for _, ad := range agentData {
		agentID := slug(ad.Name)
		// Update agent registry with discovered metadata
		s.cfg.Agents[agentID] = config.AgentDef{
			Type:         "cli",
			Command:      ad.Command,
			Path:         ad.Command,
			Capabilities: ad.Capabilities,
			Provider:     ad.Provider,
			SupportsMCP:  len(ad.MCPServers) > 0,
			DataDir:      ad.DataDir,
			Version:      ad.Version,
			Config:       ad.Config,
		}

		// Ingest discovered sessions
		for _, sess := range ad.Sessions {
			existing, err := s.store.GetSession(r.Context(), sess.ID)
			if err == nil && existing != nil {
				continue // skip existing
			}
			status := sess.Status
			if status == "" {
				status = "completed"
			}
			cruxSess := &store.Session{
				ID:        sess.ID,
				AgentID:   agentID,
				Project:   sess.Project,
				RepoPath:  sess.RepoPath,
				Status:    status,
				StartedAt: sess.StartedAt,
				Summary:   sess.Summary,
			}
			if sess.EndedAt != nil {
				cruxSess.EndedAt = sess.EndedAt
			}
			_ = s.store.CreateSession(r.Context(), cruxSess)
		}

		// Ingest discovered MCP servers
		for _, mcp := range ad.MCPServers {
			if mcp.Name == "" {
				continue
			}
			s.cfg.MCP.Servers[mcp.Name] = config.MCPServer{
				Transport: mcp.Transport,
				Command:   mcp.Command,
				Args:      mcp.Args,
				URL:       mcp.URL,
			}
		}
	}
	_ = s.cfg.Save(config.ConfigPath())

	respondJSON(w, map[string]interface{}{
		"agents":    found,
		"mcp":       discovery.DiscoverMCP(r.Context()),
		"agent_data": agentData,
	})
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.store.ListAgents(r.Context())
	if err != nil {
		logger.Error("list agents failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, agents)
}

func (s *Server) handleRunAgent(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	var req struct {
		Repo    string   `json:"repo"`
		Env     []string `json:"env"`
		Session string   `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Session == "" {
		req.Session = fmt.Sprintf("sess_%d", time.Now().Unix())
	}
	workDir := discovery.WorkingDir(req.Repo)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	_, err := s.runner.StartSession(ctx, agentID, req.Session, workDir, req.Env)
	if err != nil {
		logger.Error("run agent failed", "agent", agentID, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("session started", "session", req.Session, "agent", agentID)
	respondJSON(w, map[string]interface{}{
		"session_id": req.Session,
		"agent_id":   agentID,
		"status":     "running",
		"repo":       workDir,
	})
}

func (s *Server) handleSessionInput(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var req struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	handle, _, ok := s.runner.GetHandle(sessionID)
	if !ok {
		http.Error(w, "session not found or not active", http.StatusNotFound)
		return
	}
	if err := s.runner.SendInput(r.Context(), sessionID, handle, []byte(req.Data)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]string{"status": "sent"})
}

func (s *Server) handleSessionAttach(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	handle, br, ok := s.runner.GetHandle(sessionID)
	if !ok {
		http.Error(w, "session not found or not active", http.StatusNotFound)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer ws.Close()
	logger.Info("websocket attached", "session", sessionID)

	stdoutCh := br.Subscribe()
	defer func() { _ = stdoutCh }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for data := range stdoutCh {
			if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		}
	}()

	go func() {
		for {
			mt, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage || mt == websocket.BinaryMessage {
				_ = s.runner.SendInput(r.Context(), sessionID, handle, data)
			}
		}
	}()

	<-done
	logger.Info("websocket detached", "session", sessionID)
}

func (s *Server) handleSessionResize(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var req struct {
		Rows uint16 `json:"rows"`
		Cols uint16 `json:"cols"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	handle, _, ok := s.runner.GetHandle(sessionID)
	if !ok {
		http.Error(w, "session not found or not active", http.StatusNotFound)
		return
	}
	if err := handle.Resize(req.Rows, req.Cols); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]string{"status": "resized"})
}

func (s *Server) handleSessionStop(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if err := s.runner.StopSession(sessionID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	end := time.Now().UTC()
	_ = s.store.UpdateSession(r.Context(), &store.Session{
		ID:      sessionID,
		Status:  "stopped",
		EndedAt: &end,
	})
	logger.Info("session stopped", "session", sessionID)
	respondJSON(w, map[string]string{"status": "stopped"})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	sessions, err := s.store.ListSessions(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, sessions)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	sess, err := s.store.GetSession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	respondJSON(w, sess)
}

func (s *Server) handleSessionLogs(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	lines, err := s.store.GetTranscript(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, lines)
}

func (s *Server) handleSessionEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	events, err := s.store.ListEvents(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, events)
}

func (s *Server) handleSessionSummary(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	sess, err := s.store.GetSession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	lines, err := s.store.GetTranscript(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("Session: %s\nAgent: %s\nStatus: %s\nStarted: %s\n\n",
		sess.ID, sess.AgentID, sess.Status, sess.StartedAt.Format(time.RFC3339)))
	if sess.Summary != "" {
		out.WriteString(fmt.Sprintf("Summary: %s\n\n", sess.Summary))
	}
	out.WriteString("Transcript:\n")
	for _, l := range lines {
		prefix := "[OUT]"
		if l.IsInput {
			prefix = "[IN]"
		}
		out.WriteString(fmt.Sprintf("%s %s\n", prefix, l.Line))
	}
	_ = s.store.UpdateSessionSummary(r.Context(), sessionID, out.String())
	respondJSON(w, map[string]string{"summary": out.String()})
}

func (s *Server) handleSessionReplay(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var req struct {
		Speed float64 `json:"speed"` // multiplier, default 1.0
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Speed <= 0 {
		req.Speed = 1.0
	}

	lines, err := s.store.GetTranscript(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build replay with artificial delays based on timestamps
	type replayLine struct {
		DelayMs int    `json:"delay_ms"`
		Line    string `json:"line"`
		IsInput bool   `json:"is_input"`
	}

	var replay []replayLine
	var lastTime *time.Time
	for _, l := range lines {
		delay := 0
		if lastTime != nil {
			d := int(l.CreatedAt.Sub(*lastTime).Milliseconds())
			if d > 5000 {
				d = 5000 // cap delay at 5s for replay
			}
			delay = int(float64(d) / req.Speed)
		}
		lastTime = &l.CreatedAt
		replay = append(replay, replayLine{
			DelayMs: delay,
			Line:    l.Line,
			IsInput: l.IsInput,
		})
	}

	respondJSON(w, map[string]interface{}{
		"session_id": sessionID,
		"speed":      req.Speed,
		"lines":      replay,
	})
}

func (s *Server) handleContinueSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var req struct {
		WithAgent string `json:"with"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sess, err := s.store.GetSession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	lines, err := s.store.GetTranscript(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	continuation := buildContinuation(sess, lines)
	newSessID := fmt.Sprintf("sess_%d", time.Now().Unix())
	workDir := sess.RepoPath
	if workDir == "" {
		workDir = "."
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	handle, err := s.runner.StartSession(ctx, req.WithAgent, newSessID, workDir, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = handle

	_ = s.runner.SendInput(r.Context(), newSessID, handle, []byte(continuation+"\n"))

	respondJSON(w, map[string]interface{}{
		"previous_session": sessionID,
		"new_session":      newSessID,
		"agent_id":         req.WithAgent,
		"continuation":     continuation,
	})
}

func (s *Server) handleMCPList(w http.ResponseWriter, r *http.Request) {
	var out []map[string]interface{}
	for name, srv := range s.cfg.MCP.Servers {
		out = append(out, map[string]interface{}{
			"name":      name,
			"transport": srv.Transport,
			"command":   srv.Command,
			"args":      srv.Args,
			"url":       srv.URL,
		})
	}
	respondJSON(w, out)
}

func (s *Server) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	// MVP: return tools from known MCP server configs
	var tools []map[string]interface{}
	for name := range s.cfg.MCP.Servers {
		tools = append(tools, map[string]interface{}{
			"name":        name + ".read",
			"server":      name,
			"description": "Tool provided by " + name,
		})
	}
	respondJSON(w, tools)
}

func (s *Server) handleMCPCalls(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	calls, err := s.store.ListMCPCalls(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, calls)
}

func (s *Server) handleMCPGenerate(w http.ResponseWriter, r *http.Request) {
	path, err := gateway.GenerateConfig(s.cfg, filepath.Join(s.cfg.DataDir, "gateway"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]string{"path": path})
}

func (s *Server) handleMCPPolicy(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, s.cfg.Policies)
}

func (s *Server) handleMCPPolicyUpdate(w http.ResponseWriter, r *http.Request) {
	var p config.Policies
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.cfg.Policies = &p
	_ = s.cfg.Save(config.ConfigPath())
	logger.Info("mcp policy updated")
	respondJSON(w, s.cfg.Policies)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.Stats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, stats)
}

func respondJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func slug(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

func buildContinuation(sess *store.Session, lines []store.TranscriptLine) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("You are continuing a coding-agent session previously handled by %s.\n\n", sess.AgentID))
	b.WriteString(fmt.Sprintf("Goal: %s\n\n", sess.Summary))
	if len(lines) > 0 {
		b.WriteString("Recent transcript:\n")
		start := len(lines) - 20
		if start < 0 {
			start = 0
		}
		for _, l := range lines[start:] {
			prefix := "[OUT]"
			if l.IsInput {
				prefix = "[IN]"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", prefix, l.Line))
		}
	}
	b.WriteString("\nContinue from here.\n")
	return b.String()
}
