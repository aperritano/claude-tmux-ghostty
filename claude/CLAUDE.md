# Global Claude Code Instructions

## Agent Teams — USE TEAMS, NOT AGENT TOOL (MANDATORY)

**CRITICAL RULE**: When you need 2+ parallel workers, you MUST use **TeamCreate** to create a team, then spawn teammates using the **Agent tool with `team_name` parameter**. This makes each teammate run in its own tmux pane.

**DO NOT** use the regular Agent tool with `run_in_background: true` for parallel tasks. That creates "Local agents" which run in-process without tmux panes.

### Correct pattern (ALWAYS use this for parallel work):
1. `TeamCreate` → creates the team
2. `Agent` tool with `team_name` + `name` params → spawns each teammate in a tmux pane
3. `TaskCreate` → create tasks for teammates
4. `SendMessage` → coordinate between teammates
5. `SendMessage` type `shutdown_request` → clean up when done

### Wrong pattern (NEVER use for parallel work):
- `Agent` tool with `run_in_background: true` ← creates invisible local agents, no tmux panes
- Multiple `Agent` tool calls without `team_name` ← runs inline, no panes

### When to use teams vs solo Agent:
- **TeamCreate + teammates**: ANY task needing 2+ parallel workers (reviews, audits, exploration, multi-file changes, research)
- **Solo Agent tool (no team)**: Only for a single quick lookup where a team is overkill

### Team naming convention:
- Team names: `pr-{number}-review`, `audit-{topic}`, `implement-{feature}`
- Teammate names: `auditor`, `verifier`, `explorer-frontend`, `explorer-backend`

---

## Execution Quality

The Deterministic Execution Protocol (DEP) governs all multi-step work.
→ Full specification: `~/.claude/rules/deterministic-execution-protocol.md`

**These four principles apply to EVERY task, regardless of size:**

1. **Read before acting** — Never assume the current state of a file, service, or system. Read it.
2. **Verify after writing** — Re-read every file immediately after writing it. Off-by-one errors, missing imports, and truncated functions are common post-write hallucinations.
3. **Never guess** — Never infer file paths, function signatures, API endpoints, config keys, or version numbers from memory. Look them up.
4. **Ground every claim** — "The spec says X" → read the spec. "The test covers Y" → read the test. Cite the source or retract the claim.

**Scale the full DEP to task complexity:**

| Task | DEP sections to apply |
|------|-----------------------|
| Single edit, quick question | §4-6 only (grounding, CoVe, uncertainty) |
| Multi-file change / refactoring | §1-6 (add problem model + step verification) |
| Multi-phase build / infra change | All §1-11 |
| Gap-finding / fix-all-issues / hardening | §11 convergence loop + §4-6 |

---

## Compaction

When compacting, always preserve the full list of modified files and any test commands.

---

## Evidence Gathering

**Never substitute static baselines for live data when tools can run.**

When authenticated tools are available (kubectl, gcloud, terraform, gh, etc.), run the real read-only commands. Never seed inventories, status reports, or audit artifacts from static files when live access exists.

**Why this matters:** Static baselines miss running services not in the hosting matrix, real drift between desired and actual state, sync failures, security coverage gaps, and live resource counts. Live commands see reality; baseline files see a frozen past moment.

**Applies to all evidence work:** service inventories, sync status, health probes, drift checks, assembly audits, cluster state, security posture. Do the work properly the first time.

