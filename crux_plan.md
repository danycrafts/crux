This is a strong positioning direction. I’d sharpen Crux around this core thesis:

> **Crux Control is the operating layer for autonomous coding agents.**
> It does not help you build one agent. It helps you discover, govern, route, observe, and coordinate many agents across vendors, teams, machines, and environments.

The Wispr Flow analogy works because the moat is not “the underlying capability exists.” The moat is **better UX + vendor abstraction + workflow ownership**. Your uploaded notes make the same point for coding agents: terminal-native agents already exist, but Crux can own the cross-agent experience, observability, fallback, and governance layer. 

I’d frame the product around four pillars:

### 1. Discover

Crux finds and inventories agents across a company:

* local CLI agents
* SDK-built custom agents
* MCP-connected agents
* Docker/Kubernetes-deployed agents
* team-owned internal agents

This answers: **“What agents exist, who owns them, and what do they depend on?”**

### 2. Govern

Crux controls what agents can do:

* model access policies
* tool allow/deny lists
* MCP server policies
* human approval workflows
* credential and secret boundaries
* audit logs

This answers: **“Can I trust these agents in production?”**

### 3. Route

Crux decides where work should go:

* Claude Code vs Codex vs Gemini CLI
* local vs cloud model
* cheap model vs expensive model
* fallback on rate limit or failure
* version comparison before promotion

This answers: **“Which agent should handle this task right now?”**

### 4. Observe

Crux records what happened:

* tool calls
* memory writes
* context usage
* model costs
* latency
* failures
* policy denials
* fallback events

This answers: **“What did the agents do, what did it cost, and what broke?”**

The cleanest one-line category definition might be:

> **Crux Control is a vendor-neutral control plane for operating autonomous coding-agent fleets.**

Or more opinionated:

> **Crux is Docker Desktop + Kubernetes control plane + Datadog for AI coding agents.**

The biggest strategic moat is not the dashboard. It is the **state, policy, and execution layer**:

* normalized session history
* portable context representation
* agent capability registry
* routing and fallback engine
* MCP/tool-call governance
* cross-vendor observability

That is where Crux becomes infrastructure rather than a wrapper.
The MVP should **not** try to be “Kubernetes for agents” immediately.

The best MVP is:

> **A local-first Crux daemon + CLI that discovers terminal coding agents, runs them through PTYs, records sessions, routes MCP traffic through agentgateway, and shows usage/cost/tool activity in one place.**

That proves the core Crux thesis without needing full enterprise governance on day one.

---

## MVP thesis

Crux should prove one thing:

> “Using Claude Code, Codex, Gemini CLI, and other coding agents through Crux is better than using them directly.”

The MVP should focus on **managed terminal agents**, because your earlier notes identify that most coding agents still operate through terminal UIs, and Crux’s value is turning that terminal chaos into a structured control plane. 

Provider SDK integrations should come later. OpenAI Agents, Claude Agent SDK, Google ADK, and Microsoft Agent Framework already help people **build** agents. Crux should instead sit above them as the **operating layer**. OpenAI’s own docs describe the agent app as owning orchestration, tool execution, approvals, and state, which maps directly to the harness/control-plane layer Crux wants to own. ([OpenAI Developers][1])

---

# MVP: 6–8 week version

Assuming a small team, the first MVP should have six components.

## 1. `crux` CLI

This is the main developer entry point.

Core commands:

```bash
crux init
crux discover
crux agents
crux run claude-code
crux run gemini
crux sessions
crux logs <session-id>
crux continue <session-id> --with codex
crux mcp list
crux mcp policy apply
```

The CLI should feel like:

```bash
docker ps
docker logs
docker exec
docker compose
```

but for agent sessions.

---

## 2. Local Crux daemon

This is the MVP “Crux Engine.”

Responsibilities:

* manage agent processes
* spawn agents in PTYs
* track sessions
* store logs
* expose local HTTP/gRPC API
* manage agentgateway config
* record basic usage/cost/tool events

Keep it local-first at the start:

```text
crux CLI
   ↓
local Crux daemon
   ↓
PTY workers / agentgateway / SQLite
```

Do **not** start with SaaS multi-tenancy. That adds too much complexity too early.

---

## 3. Agent discovery

MVP discovery should detect installed agent CLIs:

```bash
which claude
which codex
which gemini
which opencode
which aider
```

Then create an agent registry:

```yaml
agents:
  claude-code:
    type: cli
    command: claude
    capabilities:
      - code_edit
      - shell
      - mcp
    owner: local-user

  gemini-cli:
    type: cli
    command: gemini
    capabilities:
      - code_edit
      - shell
```

This immediately answers one of Crux’s core operational questions:

> “What agents exist on this machine, and how are they configured?”

Later this becomes company-wide inventory.

---

## 4. PTY-based runner

This is the heart of the MVP.

Plain stdin/stdout is not enough because many terminal agents expect a real TTY. The MVP runner should:

* spawn each agent inside a PTY
* stream input/output
* capture ANSI terminal output
* detect exit codes, crashes, timeouts
* store raw transcript
* optionally normalize transcript into structured events

Minimum event model:

```json
{
  "session_id": "sess_123",
  "agent": "claude-code",
  "event_type": "terminal_output",
  "timestamp": "...",
  "content": "..."
}
```

Later event model:

```json
{
  "event_type": "tool_call",
  "tool": "file.edit",
  "status": "allowed",
  "latency_ms": 1200,
  "cost_usd": 0.04
}
```

For MVP, raw capture is enough. Perfect parsing can come later.

---

## 5. Session store

Use SQLite locally.

Store:

* agent sessions
* terminal transcripts
* prompts
* summarized context
* working directory
* git repo metadata
* model/provider metadata where available
* failure reasons
* fallback attempts

Minimum useful commands:

```bash
crux sessions
crux logs sess_123
crux replay sess_123
crux summarize sess_123
```

This is where the moat starts. Your uploaded notes correctly identify session history, context windows, and fallback as the valuable state layer, not just UI. 

---

## 6. agentgateway integration for MCP

For MVP, Crux should not build its own MCP proxy. Use `agentgateway`.

agentgateway already supports proxying local stdio MCP servers and remote MCP servers through a config file with named targets. ([agentgateway][2])

MVP Crux should generate and manage this:

```yaml
mcp:
  port: 3000
  targets:
    - name: filesystem
      stdio:
        cmd: npx
        args:
          - -y
          - "@modelcontextprotocol/server-filesystem"
```

Then Crux adds:

```bash
crux mcp list
crux mcp tools
crux mcp calls
crux mcp policy block email.send
```

agentgateway is especially useful because it already supports MCP authorization rules using CEL expressions, including filtering unauthorized tools out of list responses. ([agentgateway][3])

That gives Crux an early governance story without building the whole policy engine from scratch.

---

# The MVP demo should look like this

A good MVP demo:

```bash
crux discover
```

Output:

```text
Found agents:
✓ Claude Code      /usr/local/bin/claude
✓ Gemini CLI       /usr/local/bin/gemini
✓ Codex            /usr/local/bin/codex

Found MCP servers:
✓ filesystem
✓ github
✓ postgres
```

Then:

```bash
crux run claude-code --repo ./my-app
```

Crux opens a managed PTY session.

Then:

```bash
crux sessions
```

Output:

```text
SESSION     AGENT        STATUS     COST     TOOLS      STARTED
sess_101    claude-code  running    $0.18    14 calls   12 min ago
sess_100    gemini-cli   exited     $0.03    3 calls    yesterday
```

Then:

```bash
crux continue sess_101 --with gemini-cli
```

Crux summarizes the previous session and injects a continuation prompt into Gemini CLI.

This is not yet perfect state portability, but it proves the value:

> “I can move work across coding agents without manually copy-pasting terminal history.”

---

# What to exclude from the MVP

Do **not** build these first:

* full cloud dashboard
* enterprise RBAC
* Kubernetes workers
* full billing system
* custom hosted agent runtime
* agent marketplace
* perfect cross-model memory
* deep SDK support for every provider
* automated production-grade fallback
* complex eval framework

Those are roadmap features.

The MVP should be boring infrastructure with one magical workflow:

> discover agents → run managed session → observe activity → resume/fallback into another agent.

---

# Roadmap

## Phase 0 — Design spike, 1–2 weeks

Goal: define the primitives.

Deliverables:

* `crux.yaml` spec
* agent registry schema
* session event schema
* PTY runner prototype
* agentgateway config generator
* SQLite schema
* first adapters: Claude Code, Gemini CLI, Codex or OpenCode

Key design decision:

```text
Crux Agent = executable + capabilities + environment + policies + session history
```

Example:

```yaml
version: 0.1

agents:
  claude-code:
    type: cli
    command: claude
    working_dir: "."
    mcp_gateway: default
    fallback:
      - gemini-cli
      - codex

policies:
  tools:
    deny:
      - email.send
      - shell.rm_recursive
    require_approval:
      - github.pr.merge
```

---

## Phase 1 — Local developer MVP, 6–8 weeks

Goal: make Crux useful for one developer.

Features:

* local daemon
* CLI
* agent discovery
* PTY execution
* session recording
* session replay
* basic continuation into another agent
* agentgateway-managed MCP proxy
* simple local dashboard or TUI
* SQLite storage
* basic cost/usage estimation where available

This version should answer:

* What agents do I have?
* What sessions are running?
* What happened in the last session?
* Which MCP tools are exposed?
* Can I block or hide a dangerous MCP tool?
* Can I continue a session with another agent?

This is the first version people can actually use.

---

## Phase 2 — Team pilot, 2–3 months

Goal: make Crux useful for a small engineering team.

Add:

* shared team server
* users/projects
* agent ownership
* project-level agent registry
* centralized session history
* human approval workflows
* policy packs
* basic cost dashboard
* OpenTelemetry export
* Docker-based workers
* GitHub/GitLab repo awareness
* manual fallback chains

This phase is where Crux starts becoming B2B.

Example policy:

```yaml
policies:
  production-repos:
    require_approval:
      - file.write
      - shell.exec
      - github.pr.create
    deny:
      - secrets.read
      - email.send
```

Claude Agent SDK already exposes concepts like permissions, hooks, sessions, MCP, cost tracking, and OpenTelemetry, so Crux can integrate with those later instead of inventing every hook from scratch. ([Claude][4])

---

## Phase 3 — SDK-built agent support, 3–5 months

Goal: expand beyond terminal agents.

Add adapters for:

* OpenAI Agents SDK
* Claude Agent SDK
* Google ADK
* Microsoft Agent Framework
* raw API agents
* custom internal agents

Google ADK supports building, debugging, deploying, evaluating, and scaling agents, including multi-agent systems and deployment to Cloud Run or GKE. ([Google Cloud Documentation][5]) Microsoft Agent Framework similarly focuses on building agents and multi-agent workflows in .NET and Python, with features such as session-based state management, telemetry, filters, and model support. ([Microsoft Learn][6])

Crux should treat these as **agent sources**, not competitors.

Adapter shape:

```ts
interface CruxAgentAdapter {
  discover(): AgentDefinition[]
  startSession(input: SessionInput): SessionHandle
  streamEvents(session: SessionHandle): AsyncIterable<CruxEvent>
  stop(session: SessionHandle): Promise<void>
}
```

This is where Crux becomes genuinely framework-agnostic.

---

## Phase 4 — Enterprise control plane, 6–9 months

Goal: make Crux valuable to managers, platform teams, and security teams.

Add:

* org/team RBAC
* SSO
* audit log
* policy-as-code
* cost attribution by team/project/user
* agent versioning
* approvals UI
* SIEM export
* compliance reports
* MCP quarantine
* secrets boundaries
* anomaly detection

This answers the B2B questions:

* Which teams are using which agents?
* What did they cost this month?
* Which tools did they call?
* Which policies were denied?
* Which MCP servers are risky?
* Which agents touched production repos?

This is the budget-owner version of Crux.

---

## Phase 5 — Real orchestration layer, 9–12+ months

Goal: move from observability/control to active coordination.

Add:

* automatic fallback
* task routing
* agent capability matching
* eval-based routing
* cost-aware model selection
* agent version comparison
* context IR
* durable memory layer
* session diffing
* rollback/checkpointing
* Kubernetes workers
* remote sandboxes

This is the “Kubernetes for agent workforces” version.

But it should come after the MVP proves that people want Crux around their daily coding agents.

---

# Product packaging

I’d package it like this:

## Crux Local

For individual developers.

* local daemon
* local session history
* agent discovery
* PTY runner
* MCP visibility
* fallback/manual continuation

This can be open source or freemium.

## Crux Team

For startups and engineering teams.

* shared server
* team dashboard
* project policies
* cost tracking
* audit logs
* approval workflows

## Crux Enterprise

For regulated companies.

* SSO/RBAC
* compliance exports
* SIEM integration
* private deployment
* Kubernetes workers
* advanced governance
* full agent inventory

---

# The real MVP feature priority

I’d rank features like this:

| Priority | Feature                | Why                                              |
| -------: | ---------------------- | ------------------------------------------------ |
|       P0 | PTY runner             | Without this, Crux cannot manage terminal agents |
|       P0 | Session recording      | This creates the state moat                      |
|       P0 | Agent discovery        | Immediate “control plane” feel                   |
|       P0 | CLI                    | Developers need scriptability                    |
|       P1 | agentgateway MCP proxy | Fast path to tool governance                     |
|       P1 | Session continuation   | First step toward fallback                       |
|       P1 | Local dashboard/TUI    | Makes value visible                              |
|       P2 | Cost tracking          | Useful, but provider-specific early              |
|       P2 | Policies               | Start simple: allow/deny/approval                |
|       P3 | SDK adapters           | Important, but not day-one                       |
|       P3 | Cloud dashboard        | B2B expansion                                    |
|       P3 | Kubernetes workers     | Later infrastructure scale                       |

---

# Suggested 90-day roadmap

## Days 1–15

Build technical foundation:

* `crux.yaml`
* local daemon
* SQLite
* PTY runner
* basic CLI
* one working agent adapter

Success criterion:

```bash
crux run claude-code
crux logs <session>
```

works reliably.

---

## Days 16–45

Build useful local product:

* discover Claude/Codex/Gemini/OpenCode
* record sessions
* list sessions
* replay logs
* summarize sessions
* continue session with another agent
* generate agentgateway config
* list MCP tools

Success criterion:

A developer can manage two or more coding agents from Crux and stop using raw terminal tabs for those sessions.

---

## Days 46–75

Build pilot-ready version:

* local web dashboard
* cost/usage approximation
* MCP call log
* basic policy rules
* approval prompt for dangerous tools
* Docker worker mode
* export logs as JSONL/OpenTelemetry

Success criterion:

A small team can use Crux to answer:

> “Who used which agent, what did it do, and what tools did it call?”

---

## Days 76–90

Polish and pilot:

* install script
* docs
* demo repo
* example policies
* team pilot onboarding
* crash recovery
* config validation
* session search

Success criterion:

You can onboard 5–10 serious developer users and observe whether Crux becomes part of their daily agent workflow.

---

# The roadmap in one sentence

Start with:

> **Crux Local: discover, run, record, observe, and continue terminal coding-agent sessions.**

Then expand into:

> **Crux Team: govern, audit, route, and compare agents across people and projects.**

Then become:

> **Crux Enterprise: the vendor-neutral control plane for autonomous agent workforces.**

[1]: https://developers.openai.com/api/docs/guides/agents?utm_source=chatgpt.com "Agents SDK | OpenAI API"
[2]: https://agentgateway.dev/docs/standalone/main/quickstart/mcp/?utm_source=chatgpt.com "MCP servers – agentgateway | Agent Connectivity Solved"
[3]: https://agentgateway.dev/docs/standalone/latest/mcp/mcp-authz/?utm_source=chatgpt.com "MCP authorization"
[4]: https://code.claude.com/docs/en/agent-sdk/overview?utm_source=chatgpt.com "Agent SDK overview - Claude Code Docs"
[5]: https://docs.cloud.google.com/gemini-enterprise-agent-platform/build/adk?utm_source=chatgpt.com "Agent Development Kit | Gemini Enterprise Agent Platform"
[6]: https://learn.microsoft.com/en-us/agent-framework/overview/?utm_source=chatgpt.com "Microsoft Agent Framework Overview"
Below is a fuller **Crux Control product + startup plan**: functional requirements, non-functional requirements, roadmap, revenue models, moat, GTM, risks, and validation metrics.

---

# 1. Product definition

## Core positioning

**Crux Control is a vendor-neutral control plane for operating autonomous coding-agent fleets.**

It is not an agent framework. It does not compete directly with OpenAI Agents SDK, Claude Agent SDK, Google ADK, or Microsoft Agent Framework. Those tools help developers **build agents**. Crux helps companies **operate, govern, route, observe, and coordinate agents** across tools, vendors, teams, and environments.

Your earlier framing is strong: the moat is not “better UI” alone. It is the combination of **cross-vendor abstraction, terminal-agent management, shared session state, observability, routing, fallback, and governance**. 

The product category could be described as:

> **AgentOps control plane for coding agents.**

Or more concretely:

> **Docker Desktop + Datadog + policy engine for AI coding agents.**

---

# 2. Functional requirements

## A. Agent discovery and inventory

### MVP requirements

Crux should discover locally installed coding agents.

Examples:

```bash
crux discover
```

Should detect:

* Claude Code
* Codex CLI
* Gemini CLI
* OpenCode
* Aider
* Continue
* custom CLI agents
* locally configured MCP servers

Crux should store discovered agents in an agent registry.

Example:

```yaml
agents:
  claude-code:
    type: cli
    command: claude
    path: /usr/local/bin/claude
    status: available
    provider: anthropic
    supports_mcp: true

  gemini-cli:
    type: cli
    command: gemini
    path: /usr/local/bin/gemini
    status: available
    provider: google
```

### Later requirements

Crux should discover:

* agents running in Docker
* agents deployed in Kubernetes
* agents built with provider SDKs
* MCP servers used by each agent
* model dependencies
* tool dependencies
* memory stores
* credentials used
* owning user/team/project

### Business value

This answers:

> “What agents exist across my company, and who owns them?”

---

## B. Agent registry

Crux needs a canonical registry of all managed agents.

### Required fields

Each agent should have:

* agent ID
* display name
* type: `cli`, `sdk`, `remote`, `container`, `kubernetes`
* provider: `OpenAI`, `Anthropic`, `Google`, `local`, `custom`
* command or endpoint
* version
* owner
* team
* project
* capabilities
* model used
* tools available
* MCP servers attached
* memory backends
* policy profile
* cost profile
* fallback chain
* status

Example:

```json
{
  "id": "agent_claude_code_local",
  "name": "Claude Code Local",
  "type": "cli",
  "provider": "anthropic",
  "command": "claude",
  "capabilities": ["code_edit", "shell", "mcp", "repo_search"],
  "owner": "hafiza",
  "team": "engineering",
  "policy_profile": "default-dev",
  "fallback_chain": ["codex-local", "gemini-cli"]
}
```

### Why it matters

The registry becomes the foundation for routing, governance, usage tracking, and enterprise inventory.

---

## C. PTY-based agent execution

This is the core MVP requirement.

Most coding agents are terminal-native. Crux should run them inside pseudo-terminals rather than ordinary stdin/stdout pipes.

### MVP requirements

Crux should be able to:

```bash
crux run claude-code
crux run gemini-cli
crux run codex
```

For each session, Crux should:

* spawn the agent in a PTY
* pass user input into the PTY
* stream terminal output back to the user
* capture raw terminal output
* capture ANSI sequences where needed
* detect crashes
* detect exit codes
* detect timeouts
* store transcript
* associate session with repo/project/user

### Later requirements

Crux should support:

* terminal replay
* session snapshots
* interactive web terminal
* resumable PTY sessions
* remote PTY execution
* containerized PTY workers
* Kubernetes workers
* sandboxed execution environments

### Why it matters

This is the bridge between today’s messy terminal-agent world and Crux’s structured control plane.

Your previous architecture notes correctly identify local PTY workers as the practical execution layer for CLI agents. 

---

## D. Session management

Crux needs durable session memory.

### MVP requirements

Commands:

```bash
crux sessions
crux logs <session-id>
crux replay <session-id>
crux summarize <session-id>
crux continue <session-id> --with gemini-cli
```

Each session should store:

* session ID
* agent ID
* user
* project
* repo path
* start/end time
* status
* raw transcript
* normalized messages where possible
* tool events where visible
* error events
* fallback events
* summary
* continuation prompt

### Example session object

```json
{
  "session_id": "sess_123",
  "agent_id": "claude-code",
  "project": "crux-control",
  "status": "completed",
  "started_at": "2026-05-18T10:00:00Z",
  "ended_at": "2026-05-18T10:42:00Z",
  "cost_usd": 1.27,
  "tool_calls": 18,
  "fallbacks": 0
}
```

### Later requirements

* session diffing
* session branching
* session handoff between agents
* long-term project memory
* context compression
* vector retrieval over old sessions
* session-level audit log
* human-readable timeline

### Why it matters

This is one of Crux’s strongest moats. Vendors may own their individual agent sessions, but Crux can own **cross-agent state**.

---

## E. Context portability and continuation

Do not try to solve perfect context portability in MVP.

Instead, start with **session continuation**.

### MVP requirement

Crux should generate a continuation package:

```bash
crux continue sess_123 --with codex
```

The package should include:

* previous task goal
* repo path
* files changed
* commands run
* decisions made
* unresolved issues
* last user request
* current plan
* warnings
* suggested next action

Example injected prompt:

```text
You are continuing a coding-agent session previously handled by Claude Code.

Goal:
Refactor the authentication middleware.

Progress so far:
- Located auth middleware in src/auth/middleware.ts
- Added tests in tests/auth.middleware.test.ts
- Previous agent failed while updating token refresh logic

Important context:
- Do not change public API
- Preserve existing JWT validation behavior
- Run npm test before final answer

Continue from here.
```

### Later requirement

Build a normalized **Crux Session IR**.

Possible schema:

```json
{
  "goal": "...",
  "repo_state": "...",
  "changed_files": [],
  "tool_history": [],
  "decision_log": [],
  "open_tasks": [],
  "constraints": [],
  "compressed_context": "..."
}
```

### Why it matters

True cross-agent state portability is difficult because models, tokenizers, tool schemas, context windows, and memory systems differ. But a structured continuation layer is achievable and valuable early.

---

## F. MCP gateway integration

Crux should not build its own MCP proxy first. It should integrate with agentgateway.

agentgateway is positioned as an open-source gateway for HTTP, gRPC, LLM, MCP, and agent traffic, with routing, security, observability, and governance capabilities. ([agentgateway][1])

### MVP requirements

Crux should manage:

```bash
crux mcp list
crux mcp tools
crux mcp calls
crux mcp policy apply
```

Crux should generate agentgateway configs for:

* local stdio MCP servers
* HTTP MCP servers
* per-agent MCP profiles
* per-project MCP policies

### Example

```yaml
mcp:
  gateways:
    default:
      port: 3000
      servers:
        filesystem:
          transport: stdio
          command: npx
          args:
            - -y
            - "@modelcontextprotocol/server-filesystem"
```

### Policy examples

```yaml
policies:
  deny:
    - email.send
    - github.repo.delete
    - shell.rm_recursive

  require_approval:
    - github.pr.merge
    - database.write
    - secrets.read
```

agentgateway already provides a relevant substrate because its docs describe MCP authorization rules and tool filtering. ([agentgateway][2])

### Why it matters

This gives Crux governance early:

> “Can I block all agents from calling `email.send` without human approval?”

---

## G. Policy and governance

### MVP requirements

Start simple:

* allow tool
* deny tool
* require approval
* log only
* per-agent policy
* per-project policy

Example:

```yaml
policy_profiles:
  default-dev:
    allow:
      - file.read
      - file.write
      - shell.exec
    require_approval:
      - github.pr.create
      - database.write
    deny:
      - email.send
      - secrets.export
```

### Later requirements

* policy-as-code
* RBAC
* per-user policies
* per-team policies
* environment-based policies
* production repo restrictions
* sensitive file detection
* prompt injection protection
* secrets scanning
* quarantine risky MCP servers
* policy simulation
* policy diff before deploy

### Enterprise requirement

Crux should answer:

> “Which agents were denied access to which tools, when, and why?”

---

## H. Observability

### MVP requirements

Crux should show:

* active sessions
* historical sessions
* agent used
* project
* duration
* status
* approximate cost
* tool calls
* failures
* fallbacks
* policy denials

Commands:

```bash
crux ps
crux sessions
crux stats
crux costs
crux tools
```

Dashboard views:

* sessions table
* per-agent usage
* per-project usage
* tool-call timeline
* errors
* cost estimate

### Later requirements

* OpenTelemetry export
* Prometheus metrics
* Datadog integration
* Grafana dashboards
* SIEM export
* audit timeline
* anomaly detection
* model performance comparison
* token/cost forecasting

Claude Code already supports organizational usage, cost, and tool-activity telemetry through OpenTelemetry, which supports the idea that Crux should integrate with existing vendor telemetry where available rather than only scraping terminals. ([Claude API Docs][3])

---

## I. Cost tracking

### MVP requirements

Cost tracking should start as approximate.

Crux should track:

* session duration
* provider
* model when detectable
* estimated tokens where available
* explicit cost data where vendor exposes it
* tool-call counts
* fallback cost impact

Example output:

```text
Agent          Sessions   Est. Cost   Tool Calls   Failures
Claude Code    18         $14.82      204          2
Gemini CLI     7          $2.10       41           0
Codex          5          $6.40       88           1
```

### Later requirements

* exact token accounting
* provider billing API integrations
* budget limits
* team chargeback
* per-repo cost
* cost anomaly alerts
* recommendation engine

Example:

> “Claude Code is 37% cheaper for refactors in this repo, but Codex has fewer test-failure loops.”

That becomes decision intelligence.

---

## J. Routing and fallback

### MVP requirement

Manual fallback:

```bash
crux continue sess_123 --with gemini-cli
```

### V1 requirement

Rule-based fallback:

```yaml
fallback:
  on:
    - rate_limit
    - timeout
    - crash
  chain:
    - claude-code
    - codex
    - gemini-cli
```

### V2 requirement

Intelligent routing:

Route based on:

* task type
* repo language
* model availability
* historical success rate
* cost
* latency
* tool requirements
* policy constraints

Example:

```yaml
routing:
  refactor:
    prefer: claude-code
  test_generation:
    prefer: codex
  documentation:
    prefer: gemini-cli
  max_cost_per_session: 5.00
```

### Long-term requirement

Eval-based routing:

> “For TypeScript test generation in this repo, route to Agent B because it has the best historical pass rate per dollar.”

---

## K. SDK-built agent support

This should come after the terminal-agent MVP.

Crux should support agents built with:

* OpenAI Agents SDK
* Claude Agent SDK
* Google ADK
* Microsoft Agent Framework
* custom SDKs
* raw APIs

The reason is strategic: these SDKs are already evolving into rich agent-building frameworks. OpenAI Agents SDK includes concepts such as agents, tools, handoffs, guardrails, and tracing. ([OpenAI GitHub][4]) Claude Agent SDK exposes built-in tools, hooks, subagents, MCP, permissions, and sessions. ([Claude API Docs][5]) Google ADK supports agents, tools, multi-agent orchestration, evaluation, and deployment. ([Google GitHub][6]) Microsoft Agent Framework targets .NET and Python agents and workflows, with session-based state, telemetry, filters, and model support. ([Microsoft Learn][7])

Crux should not duplicate all of that. Crux should normalize and operate agents built with those systems.

### Adapter interface

```ts
interface CruxAgentAdapter {
  discover(): Promise<AgentDefinition[]>
  startSession(input: SessionInput): Promise<SessionHandle>
  streamEvents(session: SessionHandle): AsyncIterable<CruxEvent>
  stop(session: SessionHandle): Promise<void>
}
```

---

## L. Dashboard

### MVP dashboard

Local web UI:

* active sessions
* agent inventory
* session logs
* MCP tools
* basic cost estimates
* policy events

### Team dashboard

* users
* teams
* projects
* agents by owner
* spend by team
* tool usage
* policy violations
* fallback analytics
* agent comparison

### Enterprise dashboard

* audit log
* compliance reports
* risky MCP servers
* secrets exposure attempts
* policy coverage
* production repo activity
* SIEM export status

---

## M. Deployment

### MVP

Local daemon:

```bash
crux daemon start
```

Local SQLite:

```text
~/.crux/crux.db
```

### V1

Docker:

```bash
docker run cruxcontrol/crux-engine
```

### V2

Team server:

```text
Crux Web
Crux API
Crux Worker
Postgres
Redis
Object Storage
agentgateway
```

### V3

Enterprise:

* private cloud
* self-hosted Kubernetes
* air-gapped mode
* SSO
* SIEM
* secrets manager integration
* policy-as-code repo integration

---

# 3. Non-functional requirements

## A. Reliability

Crux must not make agent sessions less reliable than using the agents directly.

Requirements:

* PTY runner should recover from daemon restarts where possible.
* Session logs should flush frequently.
* Crashes should preserve transcripts.
* Agent subprocesses should be cleaned up safely.
* Crux should never silently drop user input.
* If Crux fails, it should leave enough state for manual recovery.

Target:

```text
MVP local session log durability: < 1 second data loss
Team server API uptime target: 99.5%
Enterprise uptime target: 99.9%+
```

---

## B. Performance

Crux sits in the loop, so latency matters.

Requirements:

* PTY input/output overhead should be near-imperceptible.
* Local command startup should be fast.
* Dashboard should load active sessions quickly.
* Session log search should handle large transcripts.
* MCP proxy overhead should be minimal.

Targets:

```text
PTY stream latency: < 50ms local overhead
CLI command response: < 300ms for common commands
Dashboard active session load: < 1s
Session search: < 2s for normal projects
```

---

## C. Security

Crux will observe sensitive agent activity, so security is central.

Requirements:

* encrypt credentials at rest
* never log secrets by default
* redact known secret patterns
* support local-only mode
* support org-managed secret stores
* isolate agent permissions
* support policy enforcement before tool execution
* audit all approvals and denials
* provide tamper-evident logs for enterprise

Important security principle:

> Crux should reduce agent risk, not become a larger blast radius.

---

## D. Privacy

Developers may run agents over private codebases.

Requirements:

* local-first by default
* no cloud upload without explicit opt-in
* configurable transcript retention
* redaction before sync
* per-project privacy settings
* enterprise data residency
* ability to disable model-content collection
* audit-safe metadata-only mode

Privacy modes:

```yaml
privacy:
  mode: local_only

privacy:
  mode: metadata_only

privacy:
  mode: full_observability
```

---

## E. Compatibility

Crux must work across:

* macOS
* Linux
* WSL
* Docker
* Kubernetes later

Terminal compatibility:

* ANSI output
* interactive prompts
* terminal resizing
* Ctrl-C handling
* exit status capture
* `/dev/tty` behavior
* shell environment inheritance

Agent compatibility:

* Claude Code
* Codex CLI
* Gemini CLI
* OpenCode
* Aider
* custom shell agents
* SDK agents later

---

## F. Extensibility

Crux should be adapter-based.

Requirements:

* plugin system for agent adapters
* plugin system for telemetry exporters
* plugin system for policy engines
* plugin system for cost providers
* stable event schema
* stable registry schema
* public local API

This prevents Crux from becoming a hardcoded wrapper around today’s agents.

---

## G. Observability of Crux itself

Crux should emit its own logs and metrics.

Metrics:

* daemon uptime
* active sessions
* PTY crashes
* agent launch failures
* MCP gateway failures
* policy evaluation latency
* database write latency
* dropped events
* queue depth

---

## H. Auditability

Enterprise buyers will care deeply about this.

Requirements:

* immutable session timeline
* who started session
* which agent was used
* which model/provider was used
* which tools were exposed
* which tools were called
* which calls were denied
* which calls required approval
* who approved
* what changed in repo
* exportable audit logs

---

## I. Usability

Developers hate heavy control planes.

Requirements:

* CLI-first
* no mandatory cloud login for local mode
* works with existing agents
* does not force new workflow
* low-friction install
* readable config
* transparent logs
* easy escape hatch to raw terminal
* explain why a policy blocked something

Good UX principle:

> Crux should feel like a power tool, not surveillance software.

---

# 4. Product roadmap

## Phase 0: Technical spike

Timeline: 2 weeks

Goal: prove terminal-agent control is feasible.

Deliverables:

* PTY runner prototype
* raw session recorder
* basic `crux run`
* one working adapter, likely Claude Code or Gemini CLI
* SQLite schema
* basic event model

Success metric:

```bash
crux run claude-code
crux logs <session-id>
```

works reliably.

---

## Phase 1: Local MVP

Timeline: 6–8 weeks

Goal: useful for individual developers.

Features:

* Crux CLI
* local daemon
* agent discovery
* PTY execution
* session recording
* session replay
* session summarization
* manual continuation across agents
* agent registry
* local SQLite
* basic MCP listing
* basic agentgateway config generation
* local web dashboard or TUI

Success metric:

> 10 developers use Crux as their normal way to launch coding agents for at least one week.

---

## Phase 2: Developer Pro

Timeline: 2–3 months

Goal: make it sticky for power users.

Features:

* better dashboard
* cost estimates
* session search
* project-level memory
* reusable agent profiles
* fallback chains
* MCP tool-call logs
* basic policy engine
* approval prompts
* Docker worker mode
* OpenTelemetry export
* Git repo awareness

Success metric:

> Developers say Crux saves time because they can see, resume, compare, and switch agents.

---

## Phase 3: Team Pilot

Timeline: 3–6 months

Goal: sell to small engineering teams.

Features:

* shared Crux server
* Postgres backend
* team/project model
* users and roles
* shared session history
* cost by user/team/project
* policy packs
* approval workflows
* centralized MCP config
* audit logs
* team dashboard
* Slack/email notifications

Success metric:

> A team lead can answer: “Which agents are being used, what did they cost, and what tools did they call?”

---

## Phase 4: SDK and custom agent support

Timeline: 6–9 months

Goal: expand beyond terminal agents.

Features:

* OpenAI Agents SDK adapter
* Claude Agent SDK adapter
* Google ADK adapter
* Microsoft Agent Framework adapter
* raw API adapter
* webhook event ingestion
* custom agent registration
* normalized event schema
* normalized tool schema
* agent capability schema

Success metric:

> Crux manages both CLI agents and SDK-built agents in the same dashboard.

---

## Phase 5: Enterprise governance

Timeline: 9–12 months

Goal: enterprise-ready control plane.

Features:

* SSO/SAML/OIDC
* RBAC
* SCIM
* SIEM export
* policy-as-code
* private deployment
* Kubernetes workers
* audit exports
* compliance reports
* MCP quarantine
* budget controls
* anomaly detection
* secrets manager integration

Success metric:

> Security/platform teams use Crux to approve agent usage across engineering orgs.

---

## Phase 6: Intelligent orchestration

Timeline: 12–18 months

Goal: become the routing brain for agent fleets.

Features:

* automatic routing
* automatic fallback
* benchmark-based model choice
* task classifier
* historical success-rate routing
* eval-driven promotion
* agent version comparison
* cross-agent memory
* context IR
* multi-agent workflows
* cost/performance optimization
* agent SLOs

Success metric:

> Crux does not only observe agents; it decides which agent should do which work.

---

# 5. Startup business plan

## A. Ideal customer profiles

### ICP 1: Individual power developers

Profile:

* uses multiple coding agents
* switches between Claude Code, Codex, Gemini, Aider, etc.
* cares about session history, fallback, and productivity

Pain:

* too many terminal tabs
* no unified history
* no cost visibility
* hard to switch agents mid-task

Product:

> Crux Local / Crux Pro

---

### ICP 2: Engineering teams at AI-forward startups

Profile:

* 10–200 engineers
* many people using AI coding tools
* no central visibility
* worried about cost and tool access

Pain:

* unknown agent usage
* uncontrolled MCP tools
* duplicated spend
* no audit trail
* no standard workflow

Product:

> Crux Team

---

### ICP 3: Platform engineering teams

Profile:

* internal developer platform owners
* responsible for tooling, governance, security, and productivity

Pain:

* agents are proliferating faster than governance
* hard to standardize
* hard to support multiple vendors
* need observability and policy

Product:

> Crux Enterprise

---

### ICP 4: Regulated enterprises

Profile:

* finance
* healthcare
* insurance
* defense contractors
* legal tech
* large SaaS companies

Pain:

* cannot allow unmanaged agents to touch code, data, email, GitHub, Jira, databases, or production systems

Product:

> Self-hosted Crux Enterprise

---

## B. Revenue models

## 1. Open-source core + paid cloud

This is probably the best model.

### Free OSS

Includes:

* local daemon
* CLI
* local agent discovery
* PTY runner
* local session history
* local dashboard
* basic MCP visibility

Purpose:

* developer adoption
* trust
* ecosystem integrations
* community adapters

### Paid cloud/team

Includes:

* team dashboard
* shared history
* cost analytics
* policies
* approvals
* hosted control plane
* audit logs
* team collaboration

Why this works:

Developers adopt locally. Teams pay when they need governance and shared visibility.

---

## 2. Per-seat SaaS

Example pricing:

| Plan       |             Price | Buyer          |
| ---------- | ----------------: | -------------- |
| Free       |                $0 | individual dev |
| Pro        | $15–25/user/month | power user     |
| Team       | $30–60/user/month | startups       |
| Enterprise |            custom | large orgs     |

What users pay for:

* history retention
* dashboard
* policy engine
* cost analytics
* integrations
* team management
* audit logs

---

## 3. Usage-based pricing

Charge by:

* managed agent sessions
* logged tool calls
* retained transcripts
* telemetry volume
* number of managed agents
* policy evaluations

Example:

```text
$0.001 per managed tool call
$0.01 per recorded session
$5 per 1M telemetry events
```

This is useful for enterprise, but dangerous for developer adoption because unpredictable bills create friction.

Best approach:

> Use per-seat pricing for simplicity, with usage limits for very large enterprises.

---

## 4. Enterprise self-hosted license

For large companies:

* annual contract
* self-hosted deployment
* SSO/RBAC
* SIEM export
* private support
* compliance features
* custom integrations

Possible pricing:

```text
$30k/year small enterprise
$100k–250k/year mid-market
$500k+/year large enterprise
```

---

## 5. Marketplace revenue

Later, Crux could have a marketplace for:

* agent adapters
* MCP policy packs
* compliance templates
* eval suites
* routing policies
* enterprise connectors
* sandbox providers

Revenue:

* marketplace take rate
* certified adapter program
* premium policy packs

Do not start here. Marketplace only works after Crux has distribution.

---

## 6. Services revenue

Early enterprise customers may need help.

Offer:

* onboarding
* custom adapters
* private deployment
* policy design
* agent inventory audit
* migration from unmanaged agents

This can fund product development, but avoid becoming a consulting company.

---

# 6. Moat

## A. Cross-vendor state layer

This is the strongest technical moat.

If Crux stores:

* sessions
* context summaries
* tool history
* repo state
* decisions
* fallbacks
* outcomes

then Crux becomes the memory layer across agents.

Vendors will optimize for their own agents. Crux can optimize across all of them.

---

## B. Agent inventory graph

Over time, Crux builds a graph:

```text
User → Project → Agent → Model → Tools → MCP Servers → Costs → Outcomes
```

This becomes valuable because companies need to know:

* what exists
* who owns it
* what it can access
* what it costs
* what it changed
* whether it is safe

That graph becomes hard to replace.

---

## C. Governance data

Policy history is sticky.

Once Crux owns:

* approval workflows
* denied calls
* risk scores
* compliance exports
* audit logs

it becomes part of the company’s control environment.

That is a stronger moat than UI.

---

## D. Workflow lock-in

If teams use Crux for:

* starting agents
* approving tools
* tracking costs
* resuming sessions
* routing work
* comparing agents

then switching away means losing workflows, not just software.

---

## E. Vendor neutrality

OpenAI, Anthropic, Google, and Microsoft each have incentives to make their own stack better. Crux can be the neutral layer across them.

This matters because the current agent ecosystem is fragmenting across SDKs, CLIs, tools, MCP servers, and runtime environments. OpenAI, Anthropic, Google, and Microsoft all now have substantial agent-building surfaces, which strengthens the case for an operating layer above them rather than yet another framework inside the same layer. ([OpenAI GitHub][4])

---

## F. Local-first trust

Developers are more likely to adopt Crux if it starts local.

A local-first architecture creates trust:

* no forced cloud upload
* private code stays local
* teams can self-host
* enterprise can deploy privately

This is a strong wedge against SaaS-only competitors.

---

# 7. Competitive risks

## Risk 1: Vendors add their own dashboards

OpenAI, Anthropic, Google, and Microsoft may add better observability, cost tracking, and routing.

Mitigation:

Crux should focus on:

* cross-vendor visibility
* terminal-agent support
* policy across vendors
* MCP governance
* shared session state
* enterprise inventory

Vendors are unlikely to provide unbiased cross-vendor control.

---

## Risk 2: Better CLIs reduce need for Crux

If Claude Code, Codex, Gemini, etc. each become excellent, users may stay inside one tool.

Mitigation:

Crux should target users and companies using **multiple agents**.

The pitch is not:

> “Claude Code is bad.”

The pitch is:

> “Your company has many agents. You need one operating layer.”

---

## Risk 3: PTY scraping is brittle

Terminal UIs can change.

Mitigation:

Use three layers:

1. PTY capture for universal compatibility.
2. Official SDK/hooks/telemetry where available.
3. MCP/tool proxy logs where possible.

Claude Code’s SDK and monitoring surfaces, for example, give Crux cleaner integration options beyond terminal scraping. ([Claude API Docs][5])

---

## Risk 4: Enterprise sales cycle is slow

Mitigation:

Start developer-first with OSS/local product.

Build bottom-up adoption, then sell Team/Enterprise.

---

## Risk 5: Security concerns

Crux sees sensitive code and agent activity.

Mitigation:

* local-first
* self-hosted
* redaction
* encryption
* metadata-only mode
* transparent architecture
* SOC 2 later
* no training on customer data

---

# 8. Go-to-market strategy

## Stage 1: Developer wedge

Launch as:

> “One CLI to manage all your coding agents.”

Target communities:

* AI coding power users
* Claude Code users
* Codex users
* Gemini CLI users
* Aider/OpenCode users
* MCP developers
* platform engineers

Content ideas:

* “I replaced 6 agent terminal tabs with one control plane”
* “How to record and replay Claude Code sessions”
* “Fallback from Claude Code to Gemini CLI when rate limited”
* “MCP governance for local coding agents”
* “Docker for AI coding agents”

---

## Stage 2: Team wedge

Sell to engineering leads.

Pitch:

> “Your developers are already using coding agents. Crux gives you visibility, cost control, and guardrails without blocking them.”

Team buyer cares about:

* cost
* productivity
* governance
* consistency
* onboarding
* security

---

## Stage 3: Enterprise wedge

Sell to platform/security.

Pitch:

> “Crux gives you an enterprise control plane for autonomous agents touching code, tools, MCP servers, and internal systems.”

Enterprise buyer cares about:

* audit
* policy
* compliance
* SSO
* data residency
* risk reduction
* vendor neutrality

---

# 9. Key metrics

## Developer metrics

* weekly active developers
* sessions per developer per week
* percentage of sessions launched through Crux
* session continuation usage
* number of managed agents per user
* retention after 4 weeks
* number of agents discovered

## Team metrics

* active teams
* agents per team
* cost tracked per team
* policy events per week
* approval events
* tool calls observed
* MCP servers governed

## Business metrics

* free-to-paid conversion
* team expansion rate
* net revenue retention
* enterprise pipeline
* average contract value
* churn
* support burden per customer

## Moat metrics

* session history retained
* policies configured
* integrations connected
* agent adapters installed
* audit exports generated
* fallback chains configured
* routing rules created

---

# 10. What the MVP should prove

The MVP should prove these five things:

1. **Crux can reliably run terminal coding agents through PTYs.**
2. **Developers value unified session history.**
3. **Developers want to continue work across agents.**
4. **Teams want visibility into agent usage and cost.**
5. **MCP/tool governance is painful enough that people will pay for it.**

The MVP should not try to prove full enterprise orchestration immediately.

---

# 11. Best near-term product shape

The first shippable product should be:

```text
Crux Local
```

With:

* CLI
* daemon
* local dashboard
* agent discovery
* PTY runner
* session recorder
* session replay
* manual continuation
* MCP visibility
* basic policies
* cost estimates

Then:

```text
Crux Team
```

With:

* shared server
* team dashboard
* user/project tracking
* cost analytics
* audit logs
* approvals
* policy management

Then:

```text
Crux Enterprise
```

With:

* SSO
* RBAC
* SIEM
* self-hosting
* Kubernetes
* compliance
* advanced governance
* intelligent routing

---

# 12. One-page roadmap summary

| Phase | Product        | Main value                                   |
| ----- | -------------- | -------------------------------------------- |
| 0     | PTY prototype  | prove terminal control                       |
| 1     | Crux Local MVP | discover, run, record, replay agents         |
| 2     | Crux Pro       | session search, continuation, cost estimates |
| 3     | Crux Team      | shared visibility, policies, approvals       |
| 4     | SDK adapters   | manage CLI + SDK agents together             |
| 5     | Enterprise     | audit, RBAC, SIEM, self-hosting              |
| 6     | Orchestration  | routing, fallback, eval-based optimization   |

---

# 13. Final strategic recommendation

Do **not** position Crux as another agent framework.

Position it as:

> **The control plane for agent workforces.**

Build the MVP around:

> **Local terminal-agent management + session recording + MCP visibility + manual fallback.**

That wedge is narrow enough to build, painful enough to matter, and expandable into a much larger B2B platform.

The long-term company is not “a nice UI for Claude Code.”

The long-term company is:

> **The operating system for autonomous software-engineering agents.**

[1]: https://agentgateway.dev/?utm_source=chatgpt.com "agentgateway | Agent Connectivity Solved"
[2]: https://agentgateway.dev/docs/?utm_source=chatgpt.com "Documentation – agentgateway | Agent Connectivity Solved"
[3]: https://docs.anthropic.com/en/docs/claude-code/monitoring-usage?utm_source=chatgpt.com "Monitoring - Claude Code Docs"
[4]: https://openai.github.io/openai-agents-python/agents/?utm_source=chatgpt.com "OpenAI Agents SDK"
[5]: https://docs.anthropic.com/en/docs/claude-code/sdk/sdk-overview?utm_source=chatgpt.com "Agent SDK overview - Claude Code Docs"
[6]: https://google.github.io/adk-docs/?utm_source=chatgpt.com "Agent Development Kit (ADK) - Agent Development Kit (ADK)"
[7]: https://learn.microsoft.com/en-us/agent-framework/overview/?utm_source=chatgpt.com "Microsoft Agent Framework Overview"

