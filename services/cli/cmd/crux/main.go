package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/danycrafts/crux/pkg/logger"
	"github.com/danycrafts/crux/services/cli/internal/attach"
	"github.com/danycrafts/crux/services/cli/internal/client"
	cliconfig "github.com/danycrafts/crux/services/cli/internal/config"
	"golang.org/x/term"
)

var cfg *cliconfig.Config

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	cfg, err = cliconfig.Load(cliconfig.Path())
	if err != nil {
		cfg = cliconfig.Default()
	}

	// Override from env
	if u := os.Getenv("CRUX_API_URL"); u != "" {
		cfg.APIURL = u
	}

	// Initialize logging
	logCfg := cfg.Logging
	if logCfg.File == "" {
		logCfg.ToStdout = false // CLI only logs to stderr for errors
	}
	logger.Init(logCfg)

	c := client.New(cfg.APIURL)

	switch cmd {
	case "version":
		fmt.Println("crux 0.1.0")
	case "init":
		runInit(c)
	case "discover":
		runDiscover(c)
	case "agents":
		runAgents(c)
	case "run":
		runRun(c, args)
	case "attach":
		runAttach(c, args)
	case "sessions":
		runSessions(c, args)
	case "logs":
		runLogs(c, args)
	case "replay":
		runReplay(c, args)
	case "summarize":
		runSummarize(c, args)
	case "continue":
		runContinue(c, args)
	case "mcp":
		runMCP(c, args)
	case "config":
		runConfig(args)
	case "stats", "ps":
		runStats(c)
	case "daemon":
		runDaemon(args)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`Crux Control CLI - Operating layer for autonomous coding agents

Usage:
  crux <command> [args]

Commands:
  init                Initialize crux configuration
  discover            Discover installed agents and MCP servers
  agents              List registered agents
  run <agent>         Run an agent in a managed PTY session
  attach <session>    Attach to a running session
  sessions            List sessions
  logs <session>      Show session transcript
  replay <session>    Replay session output with timing
  summarize <session> Generate session summary
  continue <session>  Continue session with another agent
  mcp <subcommand>    MCP gateway commands (list, tools, calls, policy)
  config              View or set CLI configuration
  stats               Show aggregate stats
  daemon <action>     Start or stop the local daemon
  version             Print version
  help                Show this help

Environment:
  CRUX_API_URL        Daemon API URL (default: http://localhost:8080)
`)
}

func runInit(c *client.Client) {
	if err := c.Health(); err != nil {
		fmt.Println("Daemon not running. Starting daemon...")
		_ = startDaemon()
	}
	fmt.Println("Crux initialized.")
}

func runDiscover(c *client.Client) {
	out, err := c.Discover()
	if err != nil {
		logger.Error("discover failed", "err", err)
		os.Exit(1)
	}
	fmt.Println("Found agents:")
	agents, _ := out["agents"].([]interface{})
	for _, a := range agents {
		m, _ := a.(map[string]interface{})
		fmt.Printf("  %s\t%s\n", m["name"], m["path"])
	}
	fmt.Println("Found MCP servers:")
	mcps, _ := out["mcp"].([]interface{})
	for _, s := range mcps {
		m, _ := s.(map[string]interface{})
		fmt.Printf("  %s\n", m["name"])
	}
}

func runAgents(c *client.Client) {
	agents, err := c.ListAgents()
	if err != nil {
		logger.Error("list agents failed", "err", err)
		os.Exit(1)
	}
	fmt.Printf("%-20s %-12s %-12s %-20s %s\n", "ID", "TYPE", "PROVIDER", "COMMAND", "STATUS")
	for _, a := range agents {
		fmt.Printf("%-20s %-12s %-12s %-20s %s\n", a["id"], a["type"], a["provider"], a["command"], a["status"])
	}
}

func runRun(c *client.Client, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: crux run <agent-id> [--repo <path>]")
		os.Exit(1)
	}
	agentID := args[0]
	repo := cfg.DefaultRepo
	for i := 1; i < len(args); i++ {
		if args[i] == "--repo" && i+1 < len(args) {
			repo = args[i+1]
			i++
		}
	}
	out, err := c.RunAgent(agentID, repo, "")
	if err != nil {
		logger.Error("run failed", "err", err)
		os.Exit(1)
	}
	sessionID, _ := out["session_id"].(string)
	fmt.Fprintf(os.Stderr, "Started session %s with %s\n", sessionID, agentID)
	fmt.Fprintf(os.Stderr, "Attaching to PTY... (Ctrl-C to detach)\n")

	if term.IsTerminal(int(os.Stdin.Fd())) {
		if err := attach.Session(c, sessionID); err != nil {
			logger.Error("attach failed", "err", err)
		}
	} else {
		if err := attach.SessionNonInteractive(c, sessionID); err != nil {
			logger.Error("attach failed", "err", err)
		}
	}
}

func runAttach(c *client.Client, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: crux attach <session-id>")
		os.Exit(1)
	}
	sessionID := args[0]
	fmt.Fprintf(os.Stderr, "Attaching to session %s... (Ctrl-C to detach)\n", sessionID)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if err := attach.Session(c, sessionID); err != nil {
			logger.Error("attach failed", "err", err)
		}
	} else {
		if err := attach.SessionNonInteractive(c, sessionID); err != nil {
			logger.Error("attach failed", "err", err)
		}
	}
}

func runSessions(c *client.Client, args []string) {
	limit := 0
	for i := 0; i < len(args); i++ {
		if args[i] == "--limit" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &limit)
			i++
		}
	}
	sessions, err := c.ListSessions(limit)
	if err != nil {
		logger.Error("list sessions failed", "err", err)
		os.Exit(1)
	}
	fmt.Printf("%-16s %-16s %-12s %-8s %-10s %s\n", "SESSION", "AGENT", "STATUS", "COST", "TOOLS", "STARTED")
	for _, s := range sessions {
		fmt.Printf("%-16s %-16s %-12s %-8v %-10v %v\n", s["id"], s["agent_id"], s["status"], s["cost_usd"], s["tool_calls"], s["started_at"])
	}
}

func runLogs(c *client.Client, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: crux logs <session-id>")
		os.Exit(1)
	}
	lines, err := c.SessionLogs(args[0])
	if err != nil {
		logger.Error("logs failed", "err", err)
		os.Exit(1)
	}
	for _, l := range lines {
		prefix := "[OUT]"
		if b, ok := l["is_input"].(bool); ok && b {
			prefix = "[IN]"
		}
		fmt.Printf("%s %s\n", prefix, l["line"])
	}
}

func runReplay(c *client.Client, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: crux replay <session-id> [--speed <multiplier>]")
		os.Exit(1)
	}
	sessionID := args[0]
	speed := 1.0
	for i := 1; i < len(args); i++ {
		if args[i] == "--speed" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%f", &speed)
			i++
		}
	}
	data, err := c.SessionReplay(sessionID, speed)
	if err != nil {
		logger.Error("replay failed", "err", err)
		os.Exit(1)
	}
	lines, _ := data["lines"].([]interface{})
	for _, item := range lines {
		line, _ := item.(map[string]interface{})
		delay, _ := line["delay_ms"].(float64)
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		prefix := "[OUT]"
		if b, _ := line["is_input"].(bool); b {
			prefix = "[IN]"
		}
		fmt.Printf("%s %s", prefix, line["line"])
	}
}

func runSummarize(c *client.Client, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: crux summarize <session-id>")
		os.Exit(1)
	}
	out, err := c.SessionSummary(args[0])
	if err != nil {
		logger.Error("summarize failed", "err", err)
		os.Exit(1)
	}
	fmt.Println(out["summary"])
}

func runContinue(c *client.Client, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: crux continue <session-id> --with <agent-id>")
		os.Exit(1)
	}
	sessionID := args[0]
	withAgent := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--with" && i+1 < len(args) {
			withAgent = args[i+1]
			i++
		}
	}
	if withAgent == "" {
		fmt.Fprintln(os.Stderr, "missing --with <agent-id>")
		os.Exit(1)
	}
	out, err := c.ContinueSession(sessionID, withAgent)
	if err != nil {
		logger.Error("continue failed", "err", err)
		os.Exit(1)
	}
	fmt.Printf("Continued session %s -> %s with %s\n", sessionID, out["new_session"], withAgent)
}

func runMCP(c *client.Client, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: crux mcp <list|tools|calls|policy|generate>")
		os.Exit(1)
	}
	sub := args[0]
	switch sub {
	case "list":
		servers, err := c.MCPList()
		if err != nil {
			logger.Error("mcp list failed", "err", err)
			os.Exit(1)
		}
		for _, s := range servers {
			fmt.Printf("%s\t%s\t%s %v\n", s["name"], s["transport"], s["command"], s["args"])
		}
	case "tools":
		tools, err := c.MCPTools()
		if err != nil {
			logger.Error("mcp tools failed", "err", err)
			os.Exit(1)
		}
		fmt.Printf("%-20s %-16s %s\n", "TOOL", "SERVER", "DESCRIPTION")
		for _, t := range tools {
			fmt.Printf("%-20s %-16s %s\n", t["name"], t["server"], t["description"])
		}
	case "calls":
		sessionID := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--session" && i+1 < len(args) {
				sessionID = args[i+1]
				i++
			}
		}
		calls, err := c.MCPCalls(sessionID)
		if err != nil {
			logger.Error("mcp calls failed", "err", err)
			os.Exit(1)
		}
		fmt.Printf("%-16s %-20s %-10s %-8s %s\n", "SESSION", "TOOL", "STATUS", "COST", "TIME")
		for _, call := range calls {
			fmt.Printf("%-16s %-20s %-10s %-8v %s\n", call["session_id"], call["tool_name"], call["status"], call["cost_usd"], call["created_at"])
		}
	case "policy":
		if len(args) > 1 && args[1] == "apply" {
			fmt.Println("Policy updated (stub: use API directly for precise control).")
		} else {
			p, err := c.MCPPolicy()
			if err != nil {
				logger.Error("mcp policy failed", "err", err)
				os.Exit(1)
			}
			fmt.Printf("Deny: %v\n", p["deny"])
			fmt.Printf("Require approval: %v\n", p["require_approval"])
			fmt.Printf("Allow: %v\n", p["allow"])
		}
	case "generate":
		out, err := c.MCPGenerate()
		if err != nil {
			logger.Error("mcp generate failed", "err", err)
			os.Exit(1)
		}
		fmt.Printf("Generated config: %s\n", out["path"])
	default:
		fmt.Fprintf(os.Stderr, "unknown mcp subcommand: %s\n", sub)
	}
}

func runConfig(args []string) {
	if len(args) == 0 {
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(data))
		return
	}
	switch args[0] {
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: crux config get <key>")
			os.Exit(1)
		}
		switch args[1] {
		case "api_url":
			fmt.Println(cfg.APIURL)
		case "default_agent":
			fmt.Println(cfg.DefaultAgent)
		case "default_repo":
			fmt.Println(cfg.DefaultRepo)
		case "output_format":
			fmt.Println(cfg.OutputFormat)
		default:
			fmt.Fprintf(os.Stderr, "unknown key: %s\n", args[1])
		}
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: crux config set <key> <value>")
			os.Exit(1)
		}
		switch args[1] {
		case "api_url":
			cfg.APIURL = args[2]
		case "default_agent":
			cfg.DefaultAgent = args[2]
		case "default_repo":
			cfg.DefaultRepo = args[2]
		case "output_format":
			cfg.OutputFormat = args[2]
		default:
			fmt.Fprintf(os.Stderr, "unknown key: %s\n", args[1])
			os.Exit(1)
		}
		if err := cfg.Save(cliconfig.Path()); err != nil {
			logger.Error("save config failed", "err", err)
			os.Exit(1)
		}
		fmt.Println("Config updated.")
	default:
		fmt.Fprintf(os.Stderr, "usage: crux config [get|set]\n")
	}
}

func runStats(c *client.Client) {
	stats, err := c.Stats()
	if err != nil {
		logger.Error("stats failed", "err", err)
		os.Exit(1)
	}
	fmt.Printf("Total sessions:   %v\n", stats["total_sessions"])
	fmt.Printf("Active sessions:  %v\n", stats["active_sessions"])
	fmt.Printf("Total tool calls: %v\n", stats["total_tool_calls"])
	fmt.Printf("Total cost USD:   %v\n", stats["total_cost_usd"])
}

func runDaemon(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: crux daemon <start|stop|status>")
		os.Exit(1)
	}
	switch args[0] {
	case "start":
		if err := startDaemon(); err != nil {
			logger.Error("daemon start failed", "err", err)
			os.Exit(1)
		}
		fmt.Println("Daemon started.")
	case "stop":
		if err := stopDaemon(); err != nil {
			logger.Error("daemon stop failed", "err", err)
			os.Exit(1)
		}
		fmt.Println("Daemon stopped.")
	case "status":
		c := client.New("")
		if err := c.Health(); err != nil {
			fmt.Println("Daemon is not running.")
		} else {
			fmt.Println("Daemon is running.")
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown daemon action: %s\n", args[0])
	}
}

func startDaemon() error {
	cmd := os.Args[0]
	daemonPath := strings.Replace(cmd, "crux", "cruxd", 1)
	if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
		daemonPath = "cruxd"
	}
	p, err := os.StartProcess(daemonPath, []string{"cruxd"}, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		return err
	}
	_ = p.Release()
	return nil
}

func stopDaemon() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	pidPath := home + "/.crux/cruxd.pid"
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return err
	}
	var pid int
	fmt.Sscanf(string(data), "%d", &pid)
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(os.Interrupt)
}
