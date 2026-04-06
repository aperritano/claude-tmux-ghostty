# Deterministic Execution Protocol (DEP)

Research-backed techniques for ensuring Claude produces verified, correct results
without hallucination or plan drift. Apply to any multi-step task across any project.

**Research basis:** Model-First Reasoning (Dec 2025), Chain-of-Verification (ACL 2024),
Process Reward Models (ICLR 2025), Six Sigma Agent (Jan 2026), Agent Hallucination
Taxonomy (Sep 2025).

---

## Principle: Separate Thinking From Doing

The #1 research finding: LLMs that model a problem and solve it in the same pass
produce more constraint violations than those that model first, verify the model,
then execute against it. For any non-trivial task follow: **Model → Verify → Execute → Confirm.**

---

## 1. Model-First Reasoning (Before ANY Execution)

Before executing the first step of any multi-step task, construct an explicit problem model.
**You are prohibited from executing any actions during this modeling step.**

```
### Problem Model for [Task/Phase Name]

**Entities:**
- [List every file, service, config, resource involved]

**Initial State:**
- [Current state of each entity — read from filesystem/git/API, never assume]

**Target State:**
- [Expected state after this task completes]

**Actions (ordered):**
1. [Step] — Precondition: [what must be true]. Postcondition: [what will be true].
2. [Step] — Precondition: [what must be true]. Postcondition: [what will be true].
...

**Constraints (must hold at ALL intermediate states):**
- [List every invariant, rule, or contract that applies]
- [List project-specific constraints from CLAUDE.md, rules, or governing docs]

**Dependencies:**
- [Which actions block which other actions]
- [Which external systems must be available]
```

**Gate:** Review the model. Do preconditions chain correctly (step N's postcondition
satisfies step N+1's precondition)? Are all constraints satisfiable simultaneously?
If not, revise the model before executing.

---

## 2. Step-Level Verification (During Execution)

After completing each action in the model, perform a **step verification checkpoint**.
Do NOT proceed to the next step until this checkpoint passes.

```
### Step Verification: [Step N of M]

COMPLETED: [What was done — specific files changed, commands run, artifacts produced]
PRECONDITION MET: [Yes/No — cite evidence from filesystem read or command output]
POSTCONDITION MET: [Yes/No — verify by reading the actual result, not by trusting memory]
STATE DELTA: [What changed. What stayed the same.]
CONSTRAINT SCAN:
  - [Constraint 1]: PASS/FAIL — [evidence]
  - [Constraint 2]: PASS/FAIL — [evidence]
  - ...
PROGRESS: [N/M steps complete. Next step: {name}. Remaining blockers: {list}]
```

**On FAIL:** Stop immediately. Do not attempt the next step. Diagnose the failure
against the problem model. If the model was wrong, update it. If execution was wrong,
fix it. Re-verify before proceeding.

---

## 3. Constraint Re-Verification (At Milestones)

At every milestone, gate, or phase transition, re-scan ALL active constraints — not
just the ones relevant to the current step. Constraints can be violated by emergent
interactions between steps that each passed individually.

```
### Full Constraint Scan: [Milestone Name]

| # | Constraint | Source | Status | Evidence |
|---|-----------|--------|--------|----------|
| 1 | [invariant text] | [source doc/rule] | PASS/FAIL | [file:line or command output] |
| 2 | ... | ... | ... | ... |

BLOCKED CONSTRAINTS: [count]
ACTION: [proceed / halt and fix / escalate to user]
```

**Rule:** If ANY constraint fails at a gate, the gate fails. No exceptions. No
"we'll fix it later." Gates are not advisory — they are hard stops.

---

## 4. Grounded Verification (Never Trust Memory)

LLMs hallucinate state. After every action, **re-read the actual result** from the
filesystem, API, or tool output. Never verify from memory or from what you "just did."

| Bad (trusting memory) | Good (grounded verification) |
|---|---|
| "I wrote the file, so it exists" | `Read path/to/file` → verify content |
| "Tests passed when I ran them" | Run tests again → read actual output |
| "The PR is in draft state" | `gh pr view N --json state` → confirm |
| "The config has the right values" | `Read config/file.yaml` → verify each value |

**For code you just wrote:** Re-read the file and verify it matches your intent.
Off-by-one errors, missing imports, wrong variable names, and truncated functions
are the most common post-write hallucinations.

---

## 5. Chain-of-Verification (For Factual Claims)

When making claims about the codebase, architecture, or external state:

1. **Draft** the claim
2. **Generate** 2-3 verification questions that would catch errors in the claim
3. **Answer** each question by reading source material (files, docs, APIs)
4. **If any answer contradicts the claim:** Retract and correct before proceeding
5. **If you cannot verify:** State explicitly: "I cannot verify [claim] because [reason]"

**Apply CoVe to these high-risk claim types:**
- "This doc says..." → Read the document
- "The spec defines..." → Read the spec section
- "This endpoint exists at..." → Read the API spec or router code
- "The test covers..." → Read the test file
- "This config sets..." → Read the config file
- "The dependency provides..." → Read the dependency's interface

---

## 6. Explicit Uncertainty (Admit What You Don't Know)

**Permission to say "I don't know" is unconditional.** A wrong answer that sounds
confident is worse than an honest "I need to check."

- **Never guess** at file paths, function signatures, API endpoints, config keys,
  or version numbers. Look them up.
- **Never fabricate** document references, section numbers, schema definitions, or
  constraint text. Read the source.
- **Never assume** a previous step succeeded. Verify it.
- **When uncertain between two approaches:** State both, explain the trade-off,
  and ask the user rather than picking one silently.

---

## 7. State Tracker as Execution Contract

When executing multi-step tasks, maintain a state tracker. It is not just a display
artifact — it is the **execution contract**. Re-read it before each step to confirm
the world is what you expect.

### Mandatory State Tracker Updates

Update the state tracker **after every step**, not just at milestones.

```
## Execution State (updated after every step)

PHASE: [current phase or task group]
STEP: [N of M in current phase]
LAST_ACTION: [what was just completed]
LAST_VERIFIED: [what verification was performed]
CONSTRAINTS_PASSING: [N of N]
BLOCKED_BY: [nothing / list of blockers]
NEXT_ACTION: [what will happen next]
ARTIFACTS_PRODUCED: [list of files written/modified]
```

**Re-read the state tracker before starting each new step.** If the state tracker
shows unexpected values (e.g., a previous step shows as incomplete, a constraint
is failing), investigate before proceeding.

---

## 8. Consensus on Judgment Calls

When a step requires judgment (not a deterministic action), generate multiple
independent reasoning paths before deciding.

```
### Judgment: [Decision description]

**Option A:** [approach] — Reasoning: [why this works]. Risk: [what could go wrong].
**Option B:** [approach] — Reasoning: [why this works]. Risk: [what could go wrong].
**Option C:** [approach] — Reasoning: [why this works]. Risk: [what could go wrong].

**Consensus:** [Where all options agree] → proceed with confidence.
**Divergence:** [Where options disagree] → [investigate / ask user].
**Selected:** [option] — Because: [cite governing constraint or doc that resolves it].
```

**Apply consensus to:**
- Architecture and design choices
- Ambiguous requirements
- Trade-offs between competing approaches
- Any action where two reasonable engineers might disagree

---

## 9. Plan Drift Detection

After every 3 steps (or at each milestone), compare your actual execution path
against the problem model from step 1:

```
### Drift Check: [Milestone]

PLANNED STEPS: [list from model]
ACTUAL STEPS: [what actually happened]
UNPLANNED ACTIONS: [anything done that wasn't in the model — explain why]
SKIPPED ACTIONS: [anything in the model that wasn't done — explain why]
CONSTRAINT DRIFT: [any constraint added, removed, or reinterpreted during execution]
SCOPE DRIFT: [any work added or removed that wasn't in the original scope]
```

**Unplanned actions require justification.** If you did something not in the model,
explain why the model was incomplete. If you can't justify it, revert.

**Skipped actions require confirmation.** If you skipped something from the model,
this is a scope change — confirm with the user before continuing.

---

## 10. Post-Task Retrospective (Self-Correction)

At the end of each major phase or task, perform a brief retrospective:

```
### Retrospective: [Phase/Task Name]

WHAT WENT AS PLANNED: [steps that matched the model]
WHAT DIVERGED: [steps that didn't — with justification]
WHAT I WOULD DO DIFFERENTLY: [if re-running this task]
HALLUCINATION RISK: [any claims I made that I'm less than 90% confident about]
VERIFICATION DEBT: [anything I should have verified but didn't]
```

**If HALLUCINATION RISK or VERIFICATION DEBT is non-empty:** Resolve it before
declaring the task complete. Go read the file. Run the command. Verify the claim.

---

## 11. Self-Correcting Convergence Loop

When a task requires finding and fixing all gaps (missing implementations, failing
tests, constraint violations, stubs, unimplemented spec items), use this iterative
loop instead of a single pass. Each iteration reveals gaps hidden behind previous ones.

```
### Convergence Loop: [Task Name]

ITERATION: [N]
MAX_ITERATIONS: [5 — adjust per task complexity]

─── STEP 1: SCAN ───
Run a comprehensive gap scan. Use project-appropriate tools:
- Compile/build errors
- Failing tests
- grep for TODO/FIXME/STUB/HACK markers
- Unimplemented interface methods or spec items
- Constraint violations (from Section 3 scan)
- Empty functions, hardcoded returns, missing error handling

GAP_LIST:
| # | Gap | Type | Severity | File:Line |
|---|-----|------|----------|-----------|
| 1 | ... | ...  | ...      | ...       |

GAP_COUNT: [total]

─── STEP 2: PRIORITIZE ───
Order by dependency — fix foundations before things that depend on them.
Group related gaps that should be fixed together.

─── STEP 3: IMPLEMENT ───
Fix gaps in priority order. Apply Section 2 (step verification) to each fix.

─── STEP 4: VERIFY ───
For each fix:
- Re-read the changed file (Section 4 — never trust memory)
- Run tests that cover the fix
- Confirm the gap is actually resolved, not just papered over

─── STEP 5: RE-SCAN ───
Run the EXACT SAME scan from Step 1 again. Do not skip this.
New gaps may have been revealed by fixes. Count them.

─── STEP 6: REPORT ───
ITERATION: [N]
GAPS_AT_START: [count]
GAPS_FIXED: [count]
NEW_GAPS_FOUND: [count]
GAPS_REMAINING: [count]
CONVERGING: [Yes if remaining < start, No otherwise]
```

### Exit Conditions

| Condition | Action |
|---|---|
| `GAPS_REMAINING = 0` AND all tests pass | **EXIT — task complete** |
| `GAPS_REMAINING > 0` AND `CONVERGING = Yes` | Continue to next iteration |
| `GAPS_REMAINING > 0` AND `CONVERGING = No` | **STOP — ask user.** You are creating gaps as fast as you fix them. Something is structurally wrong. |
| `ITERATION = MAX_ITERATIONS` | **STOP — ask user.** Report remaining gaps and ask whether to continue or reassess approach. |

### Why the Re-Scan Matters

Fixing gap A often reveals gap B that was hidden behind it. A single-pass "fix all
gaps" approach misses these emergent gaps. The re-scan treats your own output as
untrusted input — the same principle as Chain-of-Verification (Section 5) applied
to code instead of claims.

### Why the Convergence Check Matters

Without a halting condition, the loop can run forever — creating as many new gaps
as it fixes. The convergence check (`remaining < start`) catches this pattern early.
If the gap count is flat or increasing, the approach itself is flawed and needs
human judgment, not more iterations.

---

## When to Apply This Protocol

| Task complexity | Apply |
|---|---|
| Single file edit, quick question | Sections 4-6 only (grounding, CoVe, uncertainty) |
| Multi-file change, refactoring | Sections 1-6 (add model + step verification) |
| Multi-phase skill, build pipeline, infra change | All 11 sections |
| Gap-finding / fix-all-issues / hardening | Section 11 convergence loop + Sections 4-6 |
| Subagent / delegated work | Simplified: read before acting, verify after writing, admit uncertainty, check constraints |

