# Architecture

This is the single source of truth for shared contracts in this system. If you're
building a feature that depends on another package, check here first. If the
interface you need doesn't exist yet, propose it here (via PR) before writing
implementation code against something ad-hoc.

**Rule:** nobody builds business logic directly against another package's internals.
Everyone builds against the interfaces defined here. That's what lets auth, security,
email, and upload get built in parallel.

---

## 1. High-Level Flow

```
                        ┌──────────────┐
   HTTP Request ──────▶ │  Gin Router  │
                        └──────┬───────┘
                               │
                    ┌──────────┴───────────┐
                    │  security middleware │  (JWT verify, RBAC guard, rate limit)
                    └──────────┬───────────┘
                               │
                 ┌─────────────┼─────────────┬──────────────┐
                 ▼             ▼             ▼              ▼
             auth.Handler  upload.Handler  admin.Handler  (future)
                 │             │
                 ▼             ▼
           auth.Service   upload.Service
                 │             │
        ┌────────┼───────┐     ▼
        ▼        ▼       ▼  Cloudinary
   user.Repo  security.  email.Emitter
              TokenSvc        │
                               ▼
                        email.Worker (async, EventEmitter-style)
                               │
                               ▼
                        email_logs table (state machine:
                        pending → sent → delivered → opened → clicked)
```

Core principle: **handlers depend on service interfaces, services depend on repository
interfaces.** Nothing reaches past its own layer.

---

## 2. Shared Domain Model — `internal/user`

Owned jointly; changes here require review + a heads-up in the team channel,
since almost every package depends on it.

```go
// internal/user/model.go
package user

import "time"

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID               string
	FirstName		 string
	LastName		 string
	Email            string
	PasswordHash     string // empty if OAuth-only account
	Role             Role
	IsEmailVerified  bool
	AvatarUrl		 string
	FailedLoginCount int
	LockedUntil      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
```

```go
// internal/user/repository.go
package user

import "context"

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*User, error)
	Update(ctx context.Context, u *User) error
	IncrementFailedLogin(ctx context.Context, id string) (int, error)
	ResetFailedLogin(ctx context.Context, id string) error
	SetLock(ctx context.Context, id string, until time.Time) error
}
```

Anyone needing user data (auth, admin, email personalization) depends on
`user.Repository`, never on a raw SQL query against the `users` table directly.

---

## 2a. Database Schema (finalized — team agreed, ready to implement)

This is the agreed entity design. **These are decisions, not code** — the actual
`.sql` migration files are a separate, claimable task (see `CONTRIBUTING.md` §1).
Whoever picks that task up implements exactly this shape; they don't redesign it.

[DATABASE ARCHITECTURE](./rbac-db-arc.png)

### Entities

**`users`** — core identity. `password_hash` nullable (OAuth-only users won't have
one). `failed_login_attempts` / `locked_until` live here as the fast-path fields the
lockout guard reads on every login — `login_attempts` below is the durable audit
trail, not the source of truth for the lockout decision.

```
users
  id, email (unique), password_hash (nullable),
  first_name, last_name, avatar_url,
  is_email_verified (bool), failed_login_attempts (int, default 0),
  locked_until (timestamp, nullable), created_at, updated_at
```

**`roles`** + **`user_roles`** — many-to-many join table. Agreed over a simple enum
column to future-proof beyond `user`/`admin` — the RBAC guard reads through the join.

```
roles
  id, name (unique: 'user', 'admin'), description

user_roles
  user_id (FK), role_id (FK), assigned_at
  PK (user_id, role_id)
```

**`oauth_accounts`** — federation, separate from `users` so multiple providers are
just more rows, not more columns.

```
oauth_accounts
  id, user_id (FK), provider ('google'), provider_user_id, created_at
  UNIQUE (provider, provider_user_id)
```

**`refresh_tokens`** — what makes rotation + single-use enforceable.

```
refresh_tokens
  id, user_id (FK), token_hash (unique), expires_at,
  revoked (bool), replaced_by_token_id (nullable, FK → refresh_tokens.id),
  created_at
  ON DELETE CASCADE (user_id) — no reason to keep a deleted user's tokens
```

**`otp_codes`** — one table, `purpose` enum, covers magic login + email verification
+ password reset. (Staying in Postgres — no Redis for now, revisit later if the team
wants the infra experience.)

```
otp_codes
  id, user_id (FK), code_hash, purpose ('login' | 'verify_email' | 'password_reset'),
  expires_at, used_at (nullable), created_at
  ON DELETE CASCADE (user_id)
```

**`login_attempts`** — audit trail feeding the lockout logic and enumeration
detection. **`user_id` is nullable** so failed attempts against emails that don't
exist yet can still be logged — this is what makes email-enumeration attacks
detectable, not just brute-force on known accounts.

```
login_attempts
  id, user_id (nullable, FK), attempted_email, ip_address,
  success (bool), attempted_at
  ON DELETE SET NULL (user_id) — keep the row for analytics, drop the identity
```

**`email_logs`** — state machine ledger (`pending → sent → delivered → opened →
clicked → bounced/failed`) driving retry/backoff and the analytics dashboard.
**`user_id` is nullable and `ON DELETE SET NULL`** — deleting a user shouldn't erase
historical email analytics, it just anonymizes the row.

```
email_logs
  id, user_id (nullable, FK), recipient_email, email_type,
  status ('pending'|'sent'|'delivered'|'opened'|'clicked'|'bounced'|'failed'),
  provider_message_id, retry_count, error_message,
  sent_at, delivered_at, opened_at, clicked_at, created_at
  ON DELETE SET NULL (user_id)
```

**`file_uploads`** — Cloudinary metadata per uploaded file.

```
file_uploads
  id, user_id (FK), cloudinary_public_id, url, file_type,
  size_bytes, original_filename, created_at
  ON DELETE CASCADE (user_id)
```

### Relationships at a glance

```
users 1—* user_roles *—1 roles
users 1—* oauth_accounts       (CASCADE)
users 1—* refresh_tokens       (CASCADE)
users 1—* otp_codes            (CASCADE)
users 1—* login_attempts       (SET NULL — audit survives)
users 1—* email_logs           (SET NULL — audit survives)
users 1—* file_uploads         (CASCADE)
```

### Open for the migrations PR to confirm, not redecide

- Index list (at minimum: `users.email`, `login_attempts.attempted_email`,
  `email_logs.recipient_email`, every FK column) — standard indexing, not a design
  question, but worth a checklist in that PR's description.
- Column-level constraints (e.g. `CHECK` on enum-like string columns vs. Postgres
  native `ENUM` type) — implementation detail, pick whichever the person doing the
  migrations is more comfortable maintaining.

---

## 3. Auth — `internal/auth`

**Owns:** register, login (password + OAuth + magic OTP), email verification,
password reset.

```go
// internal/auth/service.go
package auth

import "context"

type Service interface {
	RegisterWithPassword(ctx context.Context, email, password string) error
	LoginWithPassword(ctx context.Context, email, password string) (*TokenPair, error)
	LoginWithGoogle(ctx context.Context, googleToken string) (*TokenPair, error)
	RequestMagicOTP(ctx context.Context, email string) error
	VerifyMagicOTP(ctx context.Context, email, otp string) (*TokenPair, error)
	VerifyEmail(ctx context.Context, token string) error
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
```

`auth.Service` depends on `user.Repository` and `security.TokenService` — it does not
implement JWT signing itself. It also depends on `email.Emitter` to fire
`user.registered`, `user.password_reset_requested`, etc., events — it does not send
email directly.

---

## 4. Security — `internal/security`

**Owns:** JWT issuance/verification, refresh token rotation, account lockout, RBAC
middleware.

```go
// internal/security/token.go
package security

import "context"

type TokenService interface {
	GenerateAccessToken(userID string, role string) (string, error)   // 15 min expiry
	GenerateRefreshToken(userID string) (string, error)               // 7 day, single-use
	VerifyAccessToken(token string) (*Claims, error)
	RotateRefreshToken(ctx context.Context, oldToken string) (*TokenPair, error)
	RevokeAllForUser(ctx context.Context, userID string) error        // on logout/password change
}

type Claims struct {
	UserID string
	Role   string
}
```

```go
// internal/security/lockout.go
package security

import "context"

const (
	MaxFailedAttempts = 5
	LockoutDuration    = 15 // minutes
)

type LockoutService interface {
	RecordFailure(ctx context.Context, userID string) (locked bool, err error)
	IsLocked(ctx context.Context, userID string) (bool, error)
	Reset(ctx context.Context, userID string) error
}
```

```go
// internal/security/rbac.go
package security

import "github.com/gin-gonic/gin"

// Roles is route-protection middleware. Usage:
//   router.GET("/admin/users", security.Roles("admin"), adminHandler.ListUsers)
func Roles(allowed ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// pulls Claims from context (set by an earlier AuthRequired middleware),
		// checks claims.Role against allowed, aborts with 403 if not permitted
	}
}

// AuthRequired verifies the access token and sets Claims in gin.Context.
func AuthRequired(ts TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extract Bearer token, ts.VerifyAccessToken, set in context or abort 401
	}
}
```

Middleware order on protected routes is always:
`AuthRequired` → `Roles(...)` → handler.

---

## 5. Email — `internal/email`

**Owns:** async event-driven sending, delivery tracking, retry/backoff, analytics.

```go
// internal/email/emitter.go
package email

type Event string

const (
	EventUserRegistered       Event = "user.registered"
	EventPasswordResetRequest Event = "user.password_reset_requested"
	EventAccountLocked        Event = "user.account_locked"
	EventEmailVerified        Event = "user.email_verified"
)

// Emitter is what other packages depend on. auth.Service calls Emit and
// moves on — it never waits for the email to actually send.
type Emitter interface {
	Emit(event Event, payload map[string]any)
}
```

```go
// internal/email/sender.go
package email

import "context"

// Sender is the low-level dispatch interface — swappable (SMTP, SES, Postmark...)
// without touching the worker or templates.
type Sender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}
```

```go
// internal/email/repository.go
package email

import "context"

type Status string

const (
	StatusPending   Status = "pending"
	StatusSent      Status = "sent"
	StatusDelivered Status = "delivered"
	StatusOpened    Status = "opened"
	StatusClicked   Status = "clicked"
	StatusFailed    Status = "failed"
)

type LogEntry struct {
	ID        string
	Recipient string
	Event     Event
	Status    Status
	Attempts  int
}

type Repository interface {
	Create(ctx context.Context, entry *LogEntry) error
	UpdateStatus(ctx context.Context, id string, status Status) error
	IncrementAttempts(ctx context.Context, id string) error
}
```

Flow: `Emitter.Emit()` → in-process event bus → `worker.go` picks it up async →
renders template → calls `Sender.Send` (wrapped in exponential backoff) →
writes/updates a `LogEntry` row at each state transition.

**Whoever builds email templates does not need the Cloudinary/SMTP integration
finished** — build against `Sender` interface with a fake/logging sender first.

---

## 6. Upload — `internal/upload`

**Owns:** file ingestion, Cloudinary streaming.

```go
// internal/upload/interface.go
package upload

import (
	"context"
	"mime/multipart"
)

// Provider is the vendor-agnostic contract. Cloudinary is the first
// implementation — this interface is what keeps it swappable later.
type Provider interface {
	Upload(ctx context.Context, file multipart.File, filename string) (url string, err error)
	Delete(ctx context.Context, publicID string) error
}
```

Uploads stream directly from Multer/Gin's in-memory buffer to Cloudinary — no temp
files written to disk. Anyone building a feature that needs file upload (e.g. admin
avatar upload) depends on `upload.Provider`, never on the Cloudinary SDK directly.

---

## 7. Admin — `internal/admin` (optional/stretch)

**Depends on:** `user.Repository` (read-only user list/search), `email.Repository`
(analytics: delivery/open/click/bounce rates). Does not implement its own data access
— it composes the two repositories above.

---

## 8. Adding or Changing a Shared Interface

1. Open an issue describing the interface change and why it's needed.
2. Propose the Go interface signature in the issue (not just prose).
3. Get a quick thumbs-up from the lead + anyone actively depending on it.
4. Open a small, standalone PR that updates this file **and** the interface
   definition in the same commit — docs and code never drift apart here.
5. Announce the merge in the team channel so in-flight PRs can rebase if needed.

---

## 9. Package Dependency Rules (enforced in review)

- `auth` → depends on `user`, `security`, `email`. Never touches `email` internals
  or `security`'s DB tables directly.
- `security` → depends on `user` (to read role for claims). No dependency on `auth`
  or `email`.
- `email` → depends on nothing outside itself except what's passed in as payload data.
- `upload` → depends on nothing else; fully standalone, as per the doc's "no vendor
  lock-in" goal.
- `admin` → depends on `user` and `email` (read paths only).

If a PR introduces a dependency arrow not listed above, that's a review blocker —
raise it as an architecture discussion first.