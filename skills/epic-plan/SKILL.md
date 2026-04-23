---
name: epic-plan
description: Plan an epic through structured interviews, recursively splitting into tasks using the `ae` CLI
disable-model-invocation: true
user-invocable: true
arguments: "<description>"
---

# Epic Plan Skill

Plan a full epic: create it, interview the user to flesh out the work, split
it into sub-tasks where needed, and finalize each leaf with enough detail
that a separate implementation session (`/epic-work`) can execute it without
further questions.

All planning state lives in `ae`. This skill uses `ae` commands for every
write so the user can exit and resume mid-plan.

Do not implement anything. Planning ends with every leaf having a finalized
body and a handoff context.

---

## Flow: `/epic-plan <description>`

1. **Resolve the epic** — `ae epics` first. If the description matches an
   existing in-progress epic, offer to resume it. Otherwise propose a slug
   (lowercase, `[a-z][a-z0-9-]*`), confirm, then `ae task:new-epic <slug>`.
2. **Plan the root task** via the per-task loop below.
3. **Recurse depth-first** into any children created by splits — finish
   child `:1` and its entire subtree before starting `:2`.
4. **Review each branch** once all its descendants are planned.
5. **Confirm completion** when the whole tree is planned.

---

## Per-task planning loop

Run this on every task, starting with the root.

1. **Draft a short seed.** Based on available context — the user's
   description for the root, or the parent body plus ancestor contexts
   (`ae task:context:get <parent>`) for children — write a minimal
   seed: a title, one paragraph capturing the task's intent, and if the
   shape is obvious, a provisional outline of the work. Keep it brief;
   this is scaffolding, not the plan. `ae task:set <id> <seed>`.
2. **Scope: branch or leaf?** Ask the user explicitly, surfacing your
   read as the recommended option:
   - **Leaf** — one chunk of work, a single implementation session could
     finish it, acceptance criteria all address one concern.
   - **Branch** — multiple chunks with natural seams (distinct
     subsystems, clear ordering, parallel streams), or a single session
     couldn't reasonably finish it.

   The user's judgment wins. The scoping decision commits the interview
   to a template. It can be revisited mid-interview if the other shape
   starts looking obviously right — re-scope, restructure the body,
   continue.
3. **Restructure to the chosen template** and write it:
   `ae task:set <id> <markdown>`. Slot the seed content into the
   appropriate sections. See *Target body — leaf task* or *Target body —
   branch task (pre-split)* below.
4. **Interview into the template.** Refine the body through questions
   (see *Interview guidelines*). Rewrite the full body after each
   meaningful answer via `ae task:set` so the user sees progress.
5. **Finalize:**

   **Branch path:**
   - Confirm the `---`-separated section structure with the user.
     Remind them: the body is frozen after split.
   - Avoid accidental `---` elsewhere (code blocks, frontmatter
     examples). The tool splits on literal `---` lines with no
     markdown awareness.
   - `ae task:split <id>`.
   - **Dependencies (one-shot prompt):** ask once whether any children
     are ordered. Record with `ae task:after <child> <pred>`. Deps
     stay editable anytime with `task:after` / `task:unafter` —
     re-prompt only if structure materially changes.
   - **Branch context:** `ae task:context:set <id>` with shared info
     children will need (cross-cutting decisions, terminology,
     references). Keep it ≤ 15 lines.
   - Recurse into child `:1`, finish its entire subtree, then `:2`, etc.

   **Leaf path:**
   - Run the **Leaf review** below.
   - **Triage** findings; resolve Category A via a focused follow-up
     interview.
   - Populate **Implementation notes** with Category B findings.
   - Write the finalized body: `ae task:set <id> <markdown>`.
   - Write the handoff context: `ae task:context:set <id>`. Keep it ≤
     15 lines — what the implementer needs that isn't obvious from
     body or ancestor contexts.

---

## Interview guidelines

- Use `AskUserQuestion` for every interview question. No plain-text
  questions unless the user asks to chat.
- One question at a time. Do not batch.
- Rewrite the body after each meaningful answer via `ae task:set` so the
  user sees progress and can correct course.
- Plain, direct language. No filler.
- Acceptance criteria: concrete, testable, observable outcomes. Transform
  "it should be fast" into "p95 response time under 200ms".
- Implementation hints the user volunteers go in **Constraints** or
  **Context**, never in **Acceptance criteria**.
- NEVER end an interview phase without confirming with the user. Always
  ask "anything else before we move on?" before proceeding to finalize
  (split for branches, review for leaves).

---

## Target body — leaf task

```markdown
# <Title>

## Problem
What's wrong or missing today? Observable symptom or unmet need.

## Desired behavior
What should happen when this is done? From the user's or caller's
perspective.

## Acceptance criteria
- [ ] Concrete, testable statement
- [ ] Another concrete, testable statement

## Scope
What is explicitly NOT part of this task.

## Constraints
Known technical boundaries: must integrate with X, cannot change Y,
performance requirements, etc.

## Context
Relevant code paths, modules, prior discussion, related issues.

## Implementation notes
(Populated after leaf review — not part of the interview.)

### Software considerations
<Idiomatic patterns, design decisions, simplification opportunities.>

### Architecture considerations
<Structural impact, module boundaries, integration points.>

### Agent workflow recommendation
<Parallelization strategy, subagents vs agent teams, risk areas.
Advice for the orchestrator that runs this leaf.>
```

**Problem**, **Desired behavior**, and **Acceptance criteria** are
required. Skip any other section that isn't relevant.

---

## Target body — branch task (pre-split)

```markdown
# <Title>

<Short intro: what this branch covers, why it exists as a unit.>

---

# <Child 1 title>

<Enough content to seed a productive interview when child 1 is planned.
Full detail is not required — the child's body will be refined during its
own planning loop.>

---

# <Child 2 title>

<Child 2 seed content.>
```

Populated directly during the branch interview — each `---`-separated
section names one sub-task. The branch body is frozen after split, so
each section should name its seam clearly and give the child's interview
a meaningful starting draft.

---

## Context authoring

`ae task:context:get <leaf>` returns the composition:
epic context → parent context → terminal sibling contexts → self context.
They stack, so keep each context ≤ 15 lines.

- **Branch context** — shared info for all children in this subtree:
  cross-cutting constraints, terminology, references. Curate and update
  as child planning reveals new shared information.
- **Leaf context (planning)** — handoff seed for the implementing agent:
  what they need to know that isn't in the body or ancestor contexts.
  The implementer will overwrite this with results and unresolved issues
  after execution.

---

## Leaf review

After the user confirms the leaf interview is complete, launch THREE review
agents in parallel. Each receives:
- The body (`ae task:get <id>`)
- The composed context (`ae task:context:get <id>`)
- Read access to the codebase

### 1. Software Review (general-purpose subagent)

Flag:
- Existing patterns the implementer should be consistent with.
- Design decisions the body leaves open.
- Reusable infrastructure (helpers, middleware, base classes) the
  implementer might miss.
- Ambiguity in acceptance criteria that could produce divergent
  implementations.

### 2. Architecture Review (general-purpose subagent)

Flag:
- Which modules/layers are affected and how.
- New interfaces, contracts, schema migrations, or config changes.
- Coupling risks and architectural drift.
- Prerequisites that must exist before this work can proceed.

### 3. Agent Workflow Review (general-purpose subagent)

Advice for the orchestrator of this leaf (the agent `/epic-work` will
spawn when it picks this task):
- Parallelization potential (high/medium/low with reasoning).
- Subagents vs agent teams, and why.
- Rough decomposition sketch — not a full plan, the orchestrator decides.
- High-risk areas warranting extra review or human checkpoints.

---

## Branch review (lightweight)

Once every descendant leaf of a branch has been planned, run ONE
general-purpose subagent with:
- The branch body (frozen, pre-split snapshot)
- The branch context
- The body of each immediate child (`ae task:get <child>`)

The agent answers:
- Does the split still make sense given what the children turned into?
- Are any obvious pieces missing? → `ae task:add-child <branch>` and plan
  the new child.
- Are sibling dependencies correct, complete, and cycle-free?
- Does the branch context adequately support the children, or should it
  be extended? → `ae task:context:set`.

If the review surfaces a structural problem that can't be repaired by
adding a child or editing deps/context, `ae task:unsplit` + re-plan may
be necessary. Rare. Confirm explicitly with the user before suggesting.

---

## Triage & Resolution (leaf reviews)

Classify each finding from the three reviews:

- **A. Open decisions & ambiguities** — user input required. Design
  choices, ambiguous requirements, trade-offs with no obvious default.
- **B. Useful context for the implementer** — non-controversial
  information. Goes into **Implementation notes** in the body.
- **C. Noise** — drop.

Flow:
1. Present findings to the user, organized by category. Be explicit that
   Category A items need their input.
2. If Category A items exist → focused follow-up interview (one question
   at a time via `AskUserQuestion`, rewrite body after each answer).
   Resolutions update **Constraints**, **Acceptance criteria**, or
   **Scope** as appropriate.
3. After Category A is resolved (or if none existed), populate
   **Implementation notes** with Category B findings from all three
   reviews.
4. Write the finalized body: `ae task:set <id> <markdown>`.
5. Show the final body and confirm before writing the handoff context.

---

## `ae` command reference (subset used by this skill)

```
ae epics                                 # list epics (resume check)
ae task:new-epic <slug>                  # create a new epic
ae task:get <id>                         # read body
ae task:set <id> <markdown>              # write body (leaf only)
ae task:context:get <id>                 # read composed context
ae task:context:set <id> <markdown>      # write context (any task)
ae task:list parent=<id>                 # list immediate children
ae task:split <id>                       # split leaf on `---`
ae task:unsplit <id>                     # undo split (rare)
ae task:add-child <parent>               # add a child to a branch
ae task:after <id> <pred>                # sibling dependency edge
ae task:unafter <id> <pred>              # remove dependency edge
ae task:record <id> <text>               # append an agent note
```

Task commands return a JSON envelope `{"ok": bool, "data": ..., "error": ...}`.
Check `ok` before continuing; surface errors to the user.
