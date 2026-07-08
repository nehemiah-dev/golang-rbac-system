# Contributing Guide

Welcome. This is a learning project — the goal is to build a production-style RBAC system in Go (Gin) as a team, without stepping on each other's toes.
This document is the source of truth for how we work together. Read it before opening
your first PR.

---

## 1. How Task Assignment Works

We don't formally assign tasks. Instead:

1. Check the **Project Board** (GitHub Projects) for open issues.
2. Pick one that's unclaimed (no linked draft PR yet).
3. Open a **draft PR** immediately — even before writing code. The draft PR *is* your
   claim on that task. This tells everyone else "this one's taken" in real time.
4. Move the issue to `In Progress` on the board.
5. When ready for review, mark the PR "Ready for review" and request a review from the
   team lead.

If you start a task and get stuck or lose steam, **say so early** in the PR or whatsapp
group and close/unassign it — don't let it sit silently blocking others.

---

## 2. Repo Structure — Know Your Boundaries

```
internal/
├── auth/       → Registration, login, OAuth, magic OTP, password reset
├── security/   → JWT lifecycle, account lockout, RBAC guards/middleware
├── email/      → Event-driven email system, delivery tracking, analytics
├── upload/     → File upload module (Cloudinary integration)
├── admin/      → Admin dashboard
└── user/       → Shared user model + repository (see rules below)
```

**Rule of thumb:** one PR should touch one package. If your change spans two packages,
that's a signal to either split the PR or that the interface between them needs
discussion first — raise it before writing the code.

### Shared files are special

`internal/user/model.go`, JWT claim structs, and any shared interfaces are used by
almost everyone. Changes here need to be:
- Small, and proposed in their own PR (not bundled with feature work)
- Reviewed by the team lead
- Announced in the team group before merging, so in-flight PRs aren't blindsided

---

## 3. Before You Write Implementation Code

Check [ARCHITECTURE.md](./docs/ARCHITECTURE.md) for the agreed interfaces (e.g. `EmailSender`,
`UserRepository`, `TokenService`). If the interface you need doesn't exist yet:

1. Don't block waiting for someone else to build it.
2. Propose the interface shape in the issue/PR discussion, get quick agreement.
3. Build against that interface with a mock/stub if the real implementation isn't
   ready yet.

This is how multiple people build auth, email, and upload in parallel without
waiting on each other.

---

## 4. Branching & Naming

```
feature/<issue-number>-<short-description>
fix/<issue-number>-<short-description>
```

Examples:
- `feature/12-jwt-rotation`
- `feature/15-account-lockout`
- `fix/22-email-retry-backoff`

Always branch off the latest `main`.

---

## 5. Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(auth): add magic OTP login flow
fix(security): correct lockout window calculation
docs(readme): update setup instructions
test(email): add retry backoff unit tests
```

Format: `type(scope): short description`. Scope = the package you touched
(`auth`, `security`, `email`, `upload`, `admin`, `user`).

---

## 6. Database Migrations — Read This Twice

- Migrations live in `/migrations/`, numbered sequentially: `0001_`, `0002_`, etc.
- **Never edit a migration that has already been merged to `main`.** If you need to
  change a table, write a new migration that alters it.
- If two PRs both add a migration, whoever merges second must rename their file to the
  next available number and rebase. Check open PRs before picking your number.

---

## 7. Pull Request Checklist

Before marking a PR "Ready for review":

- [ ] PR touches one package (or the cross-cutting change was pre-agreed)
- [ ] `go vet ./...` and `golangci-lint run` pass locally
- [ ] `go test ./...` passes locally
- [ ] New env vars are added to `.env.example`
- [ ] New migrations are sequentially numbered and don't edit merged ones
- [ ] PR description says **what** changed and **why** (link the issue)
- [ ] If you touched a shared interface, you've flagged it in the team channel

CI will re-run lint/vet/test automatically. A PR can't merge until CI is green and it
has at least one approval.

---

## 8. Code Review Etiquette

- Review within 24–48 hours if you're tagged — don't let PRs go stale.
- Comment on the *why*, not just the *what*. This is a learning project; explain the
  reasoning behind a requested change.
- Disagreements about approach get resolved in the PR thread or standup — not in
  silence, and not by just merging over an open comment.
- It's fine to request changes more than once. It's not fine to approve something you
  don't understand just to unblock someone — ask questions instead.

---

## 9. Local Environment

- Run `docker-compose up` to get Postgres + the app running locally. Everyone uses the
  same setup — no "works on my machine."
- Copy `.env.example` to `.env` and fill in your own Cloudinary/SMTP test credentials.
- Run `go test ./...` before pushing, every time.

---

## 10. Communication

- Standup 2x/week in the team group: what you picked up, what's blocked,
  what's done.
- Blocked on someone else's interface or PR? Say so immediately — don't silently wait.
- Big architectural decisions (new dependency, changing an agreed interface, restructuring
  a package) go through a short discussion in the team group before a PR is opened,
  not after.

---

## TL;DR

1. Pick an unclaimed issue → open a draft PR right away → that's your claim.
2. Stay inside your package's folder; touch shared code only in its own small, flagged PR.
3. Don't edit merged migrations — add new ones.
4. Green CI + one approval to merge.
5. If you're stuck or blocked, say so loudly and early — silence is what causes real blockers.