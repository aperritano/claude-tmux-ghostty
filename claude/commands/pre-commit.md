---
description: Lint, clean, and commit staged files with language-aware tooling
model: inherit
---

# Pre-Commit: Lint, Cleanup & Commit

You are performing a pre-commit quality pass on all staged files. Follow each step in order. Stop and report if any step fails.

## Step 1: Discover & Categorize Staged Files

Run `git diff --cached --name-only --diff-filter=d` to list staged files (excluding deletions).

If there are no staged files, check for unstaged changes and untracked files with `git status -s`. If changes exist, stage them all with `git add` for each changed/new file (excluding `node_modules/`, `dist/`, `.next/`, `.env*` files, and other generated artifacts). If there are truly no changes at all, inform the user: "No changes found to commit." and stop.

Categorize each file by language/type using its path and extension:

| Category              | Pattern                | Linter             | Formatter     |
| --------------------- | ---------------------- | ------------------ | ------------- |
| TypeScript (backend)  | `src/**/*.ts`          | `eslint --fix`     | Prettier      |
| TypeScript (frontend) | `client/**/*.{ts,tsx}` | Prettier only      | Prettier      |
| Python                | `*.py`                 | `ruff check --fix` | `ruff format` |
| Shell                 | `*.sh`                 | `shellcheck`       | —             |
| SQL                   | `*.sql`                | Claude review      | —             |
| JSON / YAML / MD      | `*.{json,yml,yaml,md}` | —                  | Prettier      |

Display a summary of staged files grouped by category before proceeding.

## Step 2: Language-Aware Lint & Auto-Fix

For each category with staged files, run the appropriate linter:

**TypeScript (backend — `src/**/\*.ts`):\*\*

```bash
npx eslint --fix <files>
```

**TypeScript (frontend — `client/**/\*.{ts,tsx}`):\*\*
Skip linting (matches project's lint-staged config — Prettier only).

**Python (`*.py`):**

```bash
ruff check --fix <files> && ruff format <files>
```

If `ruff` is not installed, read each Python file and manually review for: unused imports, syntax issues, obvious bugs. Report findings and apply fixes.

**Shell (`*.sh`):**

```bash
shellcheck <files>
```

If `shellcheck` is not installed, read each shell file and review for: unquoted variables, missing error handling, common bash pitfalls. Report findings and apply fixes.

**SQL (`*.sql`):**
Read each SQL file and review for: syntax issues, missing semicolons, unsafe patterns.

After each linter runs, re-stage any modified files:

```bash
git add <modified-files>
```

## Step 3: Format

Run Prettier on all staged files that match project formatting globs:

```bash
npx prettier --write <files matching: *.ts, *.tsx, *.json, *.md, *.yml, *.yaml>
```

Re-stage any modified files:

```bash
git add <modified-files>
```

## Step 4: Dead Code Cleanup (Conservative)

For each staged `.ts`, `.tsx`, and `.py` file:

1. Read the file
2. Identify and remove ONLY:
   - **Unused imports** — imported but never referenced in the file
   - **Unused variables** — declared but never read (not function parameters)
   - **Commented-out code blocks** — old code that was commented out (NOT explanatory comments, TODOs, or documentation)
3. Do NOT: refactor, rename, add types, restructure, or "improve" anything

After edits, re-stage:

```bash
git add <modified-files>
```

## Step 5: Run Unit Tests

If any `.ts` or `.tsx` files were staged, run the test suite:

```bash
npx vitest run --reporter=verbose
```

If tests **fail**: stop here. Report the failures and do NOT proceed to commit. Tell the user to fix the failing tests first.

If tests **pass**: continue to commit.

## Step 6: Commit

1. Run `git diff --cached --stat` and `git diff --cached` to review the final staged changes
2. Run `git log --oneline -10` to study the repo's commit message style and conventions
3. Draft a commit message following the **Conventional Commits** spec (used at Google, Meta, and across open source):

   **Subject line** — `<type>(<scope>): <description>`
   - `type`: `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `chore`, `ci`, `build`, `style`
   - `scope`: optional, the module/area affected (e.g., `pipeline`, `auth`, `client`, `mcp`)
   - `description`: imperative mood ("add", not "added"), lowercase, no period, max 50 chars
   - Total subject line must be under 72 characters

   **Body** (if needed) — blank line after subject, then:
   - Explain **why** the change was made, not what changed (the diff shows that)
   - Wrap lines at 72 characters
   - Use bullet points for multiple concerns

   **Footer** (if applicable):
   - `BREAKING CHANGE: <description>` for breaking changes
   - `Refs: #<issue>` or `Closes: #<issue>` for linked issues

   **Examples:**

   ```
   fix(auth): resolve token refresh race condition

   Concurrent requests could both trigger a refresh, causing one to
   fail with 401. Added a mutex around the refresh flow so only the
   first caller refreshes and others wait for the result.

   Closes: #342
   ```

   ```
   feat(pipeline): add Kotlin import extraction to Phase 3.6
   ```

   ```
   refactor(mcp): extract shared reconnect logic into base client
   ```

4. **Present the draft commit message to the user and ask for approval or edits** before committing
5. Once approved, create the commit using a HEREDOC:

```bash
git commit -m "$(cat <<'EOF'
<approved commit message>

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

6. Run `git status` to confirm the commit succeeded
