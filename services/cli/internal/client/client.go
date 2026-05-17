package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Client talks to cruxd.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New creates a client with defaults.
func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) wsURL(path string) string {
	return strings.Replace(c.BaseURL, "http://", "ws://", 1) + path
}

func (c *Client) do(method, path string, body, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Health checks daemon health.
func (c *Client) Health() error {
	return c.do("GET", "/health", nil, nil)
}

// Discover triggers agent discovery.
func (c *Client) Discover() (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.do("POST", "/discover", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListAgents returns agents.
func (c *Client) ListAgents() ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	if err := c.do("GET", "/agents", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RunAgent starts a session.
func (c *Client) RunAgent(agentID, repo, sessionID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	body := map[string]interface{}{"repo": repo, "session_id": sessionID}
	if err := c.do("POST", fmt.Sprintf("/agents/%s/run", agentID), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AttachSession opens a WebSocket to the session PTY.
func (c *Client) AttachSession(sessionID string) (*websocket.Conn, error) {
	url := c.wsURL(fmt.Sprintf("/sessions/%s/attach", sessionID))
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return ws, nil
}

// ResizeSession notifies the daemon of a terminal resize.
func (c *Client) ResizeSession(sessionID string, rows, cols uint16) error {
	body := map[string]interface{}{"rows": rows, "cols": cols}
	return c.do("POST", fmt.Sprintf("/sessions/%s/resize", sessionID), body, nil)
}

// StopSession stops a running session.
func (c *Client) StopSession(sessionID string) error {
	return c.do("POST", fmt.Sprintf("/sessions/%s/stop", sessionID), nil, nil)
}

// SessionInput sends data to a session via HTTP.
func (c *Client) SessionInput(sessionID, data string) error {
	body := map[string]string{"data": data}
	return c.do("POST", fmt.Sprintf("/sessions/%s/input", sessionID), body, nil)
}

// ListSessions returns sessions.
func (c *Client) ListSessions(limit int) ([]map[string]interface{}, error) {
	path := "/sessions"
	if limit > 0 {
		path = fmt.Sprintf("/sessions?limit=%d", limit)
	}
	var out []map[string]interface{}
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetSession returns one session.
func (c *Client) GetSession(id string) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.do("GET", fmt.Sprintf("/sessions/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SessionLogs returns transcript lines.
func (c *Client) SessionLogs(id string) ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	if err := c.do("GET", fmt.Sprintf("/sessions/%s/logs", id), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SessionSummary returns generated summary.
func (c *Client) SessionSummary(id string) (map[string]string, error) {
	var out map[string]string
	if err := c.do("GET", fmt.Sprintf("/sessions/%s/summary", id), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SessionReplay returns replay data with timing.
func (c *Client) SessionReplay(id string, speed float64) (map[string]interface{}, error) {
	var out map[string]interface{}
	body := map[string]interface{}{"speed": speed}
	if err := c.do("POST", fmt.Sprintf("/sessions/%s/replay", id), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ContinueSession continues a session with another agent.
func (c *Client) ContinueSession(id, withAgent string) (map[string]interface{}, error) {
	var out map[string]interface{}
	body := map[string]string{"with": withAgent}
	if err := c.do("POST", fmt.Sprintf("/sessions/%s/continue", id), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MCPList lists MCP servers.
func (c *Client) MCPList() ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	if err := c.do("GET", "/mcp/servers", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MCPTools lists available MCP tools.
func (c *Client) MCPTools() ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	if err := c.do("GET", "/mcp/tools", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MCPCalls returns MCP call history.
func (c *Client) MCPCalls(sessionID string) ([]map[string]interface{}, error) {
	path := "/mcp/calls"
	if sessionID != "" {
		path = fmt.Sprintf("/mcp/calls?session=%s", sessionID)
	}
	var out []map[string]interface{}
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MCPGenerate generates gateway config.
func (c *Client) MCPGenerate() (map[string]string, error) {
	var out map[string]string
	if err := c.do("POST", "/mcp/generate", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MCPPolicy returns current policy.
func (c *Client) MCPPolicy() (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.do("GET", "/mcp/policy", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MCPPolicyUpdate updates policy.
func (c *Client) MCPPolicyUpdate(deny, requireApproval, allow []string) error {
	body := map[string]interface{}{
		"deny":             deny,
		"require_approval": requireApproval,
		"allow":            allow,
	}
	return c.do("POST", "/mcp/policy", body, nil)
}

// Stats returns aggregate stats.
func (c *Client) Stats() (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.do("GET", "/stats", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
