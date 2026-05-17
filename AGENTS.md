# Agent Context

## Environment Specifications
- **OS:** Linux Mint 22.3 x86_64
- **Host:** HP Z2 Mini G4 Workstation SBKPF,DWKSBLF,SBKPFV3
- **Kernel:** 6.17.0-20-generic
- **Uptime:** 9 hours, 28 mins
- **Packages:** 2102 (dpkg)
- **Shell:** bash 5.2.21
- **Theme:** Mint-Y-Aqua [GTK3]
- **Icons:** Mint-Y-Sand [GTK3]
- **Terminal:** /dev/pts/0
- **CPU:** Intel Xeon E-2126G (6) @ 4.500GHz
- **GPU:** NVIDIA Quadro P1000 Mobile
- **Memory:** 15838MiB

Detailed AI agent instructions and project context are maintained in:
- `GEMINI.md` (for Gemini CLI)
- `CLAUDE.md` (for Claude-based agents)


## GitHub CLI Authentication

The GitHub CLI (`gh`) tool is pre-configured with authentication. Agents can use `gh` commands directly.

## GitHub Actions

- **CI:** `ci.yml` validates version alignment, compose config, the stable green test subset, frontend smoke checks, and docs on `main` and `v*`.
- **Snapshot Publish:** `publish-snapshot.yml` publishes all repo-built GHCR images from `main` using the root `package.json` version plus `-SNAPSHOT`.
- **Release:** `release.yml` is branch-driven: pushing `vX.Y.Z` publishes GHCR images tagged `X.Y.Z` and creates or updates the matching GitHub Release with generated notes and bundled artifacts.
- **Deploy:** `deploy-prod.yml` SSHes to the prod host and runs `scripts/deploy-prod.sh`, which recreates the stack with `.env.prod`.

Operational rules:
- Never commit `.env.prod`, SSH material, or any deploy secret; workflows must read them from GitHub secrets/variables or from the remote host's persistent local files.
- Keep `README.md`, `substrate/docs/**/*`, and workflow docs aligned when changing workflow names, required secrets/variables, release artifacts, image names, or deploy steps.
- The root `substrate/package.json` version is the release source of truth. `main` publishes `X.Y.Z-SNAPSHOT`; release refs must be named `vX.Y.Z` and match that version exactly.
- Remote prod deploys assume the checkout already exists and `.env.prod` persists on disk; workflows may recreate containers but must not delete or regenerate that env file.

## NPM Registry Configuration

A `.npmrc` file is configured at `~/.npmrc` with the following registries:
- **Default Registry:** `https://registry.npmjs.org/`
- **Scoped Registry (`@<org-name>`):** `https://npm.pkg.github.com/` (GitHub Packages)

## Git Workflow

- **Git Identity:** Always read `GIT_USERNAME` and `GIT_EMAIL` from `.env` and set via `git config --global` before committing.
- **Conventional Commits:** Use `<type>(<scope>): <description>` format.
- **Automatic Push:** Stage, commit, and push immediately after modifications.
- **Commit Format:** One-liner only, no body, no `Co-authored-by:`.

## CLI Command Guidelines

- **Use sudo:** Always use `sudo` for when really necessary like in system-level operations.
- **Non-Interactive:** Use `-y`, `--yes`, or equivalent flags for all commands.
- **Container Rebuild:** Rebuild and recreate containers (`--build --force-recreate`) after implementation changes.

## Architecture & Design Principles

- **Microservice Architecture:** Independent data, API communication, independent deployment.
- **SOLID Principles:** Strictly adhere to all five principles.
- **DRY:** Never duplicate business logic.
- **OOP:** Encapsulation and composition over inheritance.

## Script Execution Guidelines

- **Temporary Location:** All discovery/analysis scripts must be created in `/tmp/` using naming `/tmp/<agent-name>-<timestamp>-<script-name>`.
- **Never Commit:** Temporary scripts must never be added to repository history.

## File Search Guidelines

- **Avoid Git-Ignored:** Never search in `node_modules`, `.venv`, `dist`, etc.
- **Respect .gitignore:** Use `rg` or `fd` or targeted glob patterns.

### Mandatory Workflows

#### 1. Source Control (Git)
- **Trunk-Based:** Commit directly to `main` and push. **No feature branches. No pull requests.**
- **Git Identity:** Read `GIT_USERNAME` and `GIT_EMAIL` from `/home/dany/Desktop/.env` and apply via `git config --global`.
- **Conventional Commits:** Single-line messages. No body. No footer. No `Co-authored-by:` trailer. Prefixes: `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `test:`, `style:`, `perf:`.
- **Atomic Auto-Push:** After any file modification: `git add <files>`, `git commit -m "..."`, `git push`.
- **GitHub CLI:** Use pre-authenticated `gh` for repo operations.

#### 2. Secrets & Credentials
- **Templates:** `.env.local.example` (dev) and `.env.prod.example` (prod) at repo root are committed. Per-service `.env.example` files do not exist.
- **Active files:** `.env.local` and `.env.prod` are gitignored. User edits them directly; `make up [MODE=prod]` never rewrites secrets.
- **Mandatory Sync:** Update the `.example` templates whenever their real siblings gain a new variable.

#### 3. Development Standards
- **Validation:** `make lint && make test` before every push.
- **Container Rebuild:** `make restart` (i.e. `make down && make up`) after service code changes.
- **CLI sudo:** Use `sudo` only when required; prefer user-level installs.
- **Non-interactive:** Use `-y` / `--yes` flags.
- **Architecture:** Microservice boundaries. SOLID. DRY. KISS. No dead code. No mock data.

#### 4. Script Handling
- Helper / throwaway scripts belong in `/tmp/`. Never commit them.
- Persistent tooling belongs in `scripts/`.

#### 5. File Search Guidelines
- Skip `node_modules`, `.venv`, `.uv-cache`, `dist`, `build`.
- Respect `.gitignore`.


---

## Workspace Summary & Core Instructions

- **Trunk-Based Development:** No feature branches, no PRs. All changes land on `main` via direct commits and immediate pushes.
- **Conventional Commits:** Strictly enforce `<type>(<scope>): <description>` one-liners.
- **Environment Management:** Use root `.env` for global secrets and repo `.env` for local secrets. Always update `.env.example`.
- **System Integrity:** Use `sudo` for system changes; use `/tmp/` for helper scripts.
- **Microservice Focus:** Maintain service boundaries and data ownership.
- **Subagent Compliance:** Dispatched agents must be instructed to read `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md`.
