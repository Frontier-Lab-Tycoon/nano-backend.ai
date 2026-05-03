---
name: submit
description: Quality checks, learning notes, commit, PR creation workflow for nano-backend.ai
user-invocable: true
---

# Submit Workflow

Post-implementation submission pipeline: quality enforcement, learning notes, commit, and PR creation.

## Parameters

- **issue** (optional): GitHub issue number (e.g., `#12` or `12`). Auto-detected from branch name if pattern `issue-\d+` exists.
- **base_branch** (optional): Target branch for PR. Defaults to `main`.

## Repository Target Rule

This workspace is the `Frontier-Lab-Tycoon/nano-backend.ai` fork. All push and PR operations for this repository MUST target the fork, not upstream.

- Required push remote: `origin` → `https://github.com/Frontier-Lab-Tycoon/nano-backend.ai.git`
- Required PR repo: `Frontier-Lab-Tycoon/nano-backend.ai`
- Upstream `seedspirit/nano-backend.ai` may be used only for fetch/compare context, never as the PR creation target.
- Before creating a PR, verify:
  ```bash
  git remote get-url origin
  gh repo view Frontier-Lab-Tycoon/nano-backend.ai --json nameWithOwner,defaultBranchRef
  gh pr list --repo Frontier-Lab-Tycoon/nano-backend.ai --head <branch> --json number,title,url,isDraft
  ```
- Always pass `--repo Frontier-Lab-Tycoon/nano-backend.ai` to `gh pr create`, `gh pr view`, and `gh pr list` during submit.

## Workflow

### Phase 1: Pre-flight

1. **Detect issue number**
   - Check if user provided `issue`
   - Otherwise extract from branch name (pattern: `issue-(\d+)`)
   - If not found, ask user (allow "none" for no linked issue)

2. **Fetch issue details** (if issue found)
   ```bash
   gh issue view <number> --json title,body,labels
   ```

3. **Review changes**
   - `git status` — see changed/untracked files
   - `git diff` and `git diff --staged` — review content
   - `git log {base_branch}..HEAD` — existing commits on branch
   - Summarize changes to user before proceeding

### Phase 2: Test Verification

**Mandatory — never skip. Must complete before quality checks.**

1. **Inventory test scenarios**: List all tests in changed packages. For each public function or endpoint, confirm both success and failure scenarios exist.

   ```markdown
   ## Test Coverage Report
   ### <package>::<file>
   - ok: <description>
   - ok: <description>
   - missing: <what's not tested>
   ```

2. **Write missing tests**: If any public function lacks a success or failure scenario, write them now.

   - **Success scenarios**: Valid input → expected output
   - **Error/edge scenarios**: Invalid input, missing resource, boundary conditions → expected error type or behavior

3. **Run tests and verify**:
   ```bash
   go test ./... -v 2>&1  # see all output
   ```
   - All tests must pass
   - Both success and failure paths must be exercised
   - If tests fail, fix and re-run (max 3 attempts)

### Phase 3: Quality Enforcement

**Mandatory — never skip.**

Run sequentially, stop on first failure:

```bash
gofmt -l .
golangci-lint run ./...
go test ./...
```

- If `gofmt` reports files, run `gofmt -w .` and stage results
- If `golangci-lint` fails, fix the issues and re-run
- If tests fail, fix and re-run
- **All three must pass before continuing**

### Phase 4: Learning Notes

**Mandatory — every PR must include learning documents.**

Create a directory `docs/learn/NNNN-<slug>/` where `NNNN` is a zero-padded sequence number and `<slug>` summarizes the PR topic. Inside, generate **separate MD files per category**. Only create files for categories that have meaningful content — skip empty categories.

#### Directory structure

```
docs/learn/NNNN-<slug>/
├── README.md              # Always created — PR summary and category index
├── code-design.md         # Code design learnings (if applicable)
├── cs.md                  # CS concepts (if applicable)
├── go.md                  # Go programming (if applicable)
└── backend-ai.md          # Backend.AI architecture (if applicable)
```

#### README.md (always created)

```markdown
# <Title matching PR topic>

PR: #{number} (or "pending" if not yet created)
Date: YYYY-MM-DD

## What was done

<1-3 bullet summary of the implementation>

## Categories

- [Code Design](./code-design.md) — only link if file exists
- [CS](./cs.md)
- [Go Programming](./go.md)
- [Backend.AI Architecture](./backend-ai.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|------------------------|
| ... | ... | ... |

## Further study

- [ ] <Topic or resource to dive deeper into>
- [ ] <Related Backend.AI code to read: path or link>
```

#### code-design.md — Code Design

Functional/OOP design patterns, SOLID principles, DI, type design, module structure, etc.

```markdown
# Code Design

## <Topic 1>
<Explanation, example code, links to relevant source files>

## <Topic 2>
...
```

#### cs.md — CS Concepts

Data structures, algorithms, networking, OS, concurrency, protocols — language-agnostic CS knowledge.

```markdown
# CS

## <Topic 1>
<Explanation, examples, references>

## <Topic 2>
...
```

#### go.md — Go Programming

Go syntax, goroutines, channels, interfaces, error handling, package design — Go-specific knowledge.

```markdown
# Go Programming

## <Topic 1>
<Explanation, code examples, official doc links>

## <Topic 2>
...
```

#### backend-ai.md — Backend.AI Architecture

Backend.AI's Manager/Agent/Storage structure, session lifecycle, API design, domain models, etc.

```markdown
# Backend.AI Architecture

## <Topic 1>
<Explanation, architecture diagrams, related code paths>

## <Topic 2>
...
```

#### Rules

- Write in **Korean** (이 문서는 학습용이므로 한국어로 작성)
- **One concept = one `##` section** — explain each concept in depth
- Keep each file focused — do not include content that belongs in another category
- Link to relevant source files, docs, or external references
- The "Further study" checklist in README.md should be actionable — specific topics, not vague
- **Skip empty categories** — do not create files for categories with no learnings in this PR, and remove them from the Categories list in README.md
- Do NOT ask the user to review the learning notes — proceed directly to commit

### Phase 5: Commit

1. **Stage changes** (including the learning doc)
   - `git add` specific files — avoid `-A`
   - Never stage `.env`, credentials, or other sensitive files

2. **Commit message**
   - Conventional commit style: `type(scope): description`
   - Types: `feat`, `fix`, `refactor`, `test`, `docs`, `ci`, `chore`, `perf`
   - Keep first line under 80 characters

3. **Create commit** — present draft message to user for approval

### Phase 6: PR Creation

1. **Push branch**
   ```bash
   git push -u origin {branch_name}
   ```

2. **Create PR**

   Use the fork repository explicitly:

   ```bash
   gh pr create \
     --repo Frontier-Lab-Tycoon/nano-backend.ai \
     --head "{branch_name}" \
     --base "{base_branch}" \
     --draft \
     ...
   ```

   If a linked issue exists, the PR body must include the issue context and how it was resolved:

   ```bash
   gh pr create --title "type(scope): description" --body "$(cat <<'EOF'
   ## Issue

   Resolves #<number>

   **Problem**: <What was wrong or missing? 1-2 sentences.>

   ## Solution

   **Approach**: <Why this approach? Key design choice and reasoning.>

   ### Changes
   - `pkg/module/file` — <what changed and why>
   - `pkg/module/file` — <what changed and why>

   ### Key Decisions
   | Decision | Why |
   |----------|-----|
   | <e.g., used newtype pattern> | <reason> |

   ## What I learned
   <1-2 sentences linking to the learning doc directory>
   See: `docs/learn/NNNN-<slug>/`

   ## Test Plan
   - [ ] <scenario: what is being verified>
   - [ ] <scenario>
   EOF
   )"
   ```

   If no linked issue:

   ```bash
   gh pr create --title "type(scope): description" --body "$(cat <<'EOF'
   ## Background
   <Why this change? What problem or need does it address?>

   ## Changes
   - `pkg/module/file` — <what changed and why>
   - `pkg/module/file` — <what changed and why>

   ### Key Decisions
   | Decision | Why |
   |----------|-----|
   | <e.g., chose X over Y> | <reason> |

   ## What I learned
   <1-2 sentences linking to the learning doc directory>
   See: `docs/learn/NNNN-<slug>/`

   ## Test Plan
   - [ ] <scenario: what is being verified>
   - [ ] <scenario>
   EOF
   )"
   ```

3. Update the learning doc's PR number if it was "pending"

4. Report PR URL to user

### Phase 7: Summary

```
Submission Complete

  PR:        #{number} - {title}
  URL:       {url}
  Branch:    {branch_name}
  Learn doc: docs/learn/NNNN-<slug>/

Quality: All passed (fmt, lint, test)
Commits: {count} commit(s)
```

## Error Handling

### Quality check failure
Fix the issue, re-run the failing check. Never suppress lint or test failures.

### No changes to commit
Report clean working tree. Nothing to submit.

### PR already exists
Report existing PR URL instead of creating a duplicate.
