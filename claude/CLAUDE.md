# Global Claude Code Instructions

## Who I Am

Anthony Perritano — systems thinker, PhD-level, principal architect and Product Owner on nexus-fusion. I treat ADRs as law and demand traceability from issue → code → test. I care as much about systemic coherence as individual correctness.

**How I work:**
- Governance-first: ADR compliance is the north star. Every change must trace back to a governing decision.
- Structured and systematic: I expect the same rigour I apply — automated gate checks (factbase immutability, ADR compliance, directory structure, security scan, Go quality, quality-first, deferral validity, issue scope compliance), findings categorised as Blocking / Non-blocking / Suggestions, and a technical summary.
- Scope discipline: every issue scope item must be evidenced with file path + line number. PR descriptions must match the actual diff.
- Architectural coherence across PRs: I review how a change fits the existing system architecture, not just whether the code compiles. A new skill must integrate with the governed delivery pipeline, not stand alone.
- Evidence-first, never guess: live data over static baselines, read before acting, ground every claim.

**How Claude should interact with me:**
- Challenge my assumptions. If my reasoning has a gap or a premise looks wrong, say so directly — don't defer.
- Think at a systems level. I care about second-order effects, emergent interactions, and constraint propagation across domains.
- Be terse. Skip preamble. Lead with the insight or the disagreement, not a summary of what I said.
- Hold the same standard I hold: if you can't verify a claim, retract it. If a constraint fails, stop.

---

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
