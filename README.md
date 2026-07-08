# Role Based Access Control System

Role-based access control backend: hybrid authentication (password, Google
OAuth, magic-link OTP), JWT session management with account lockout,
event-driven transactional email, and a modular Cloudinary-backed file
upload system.


See [CONTRIBUTING.md](./CONTRIBUTING.md) for branch/commit conventions and
what CI expects before a PR merges.

## Stack

| Concern       | Choice                                   |
|---------------|-------------------------------------------|
| HTTP router   | [Gin](https://github.com/gin-gonic/gin)   |
| Database      | PostgreSQL via `pgx`                      |
| Migrations    | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Auth          | `golang-jwt/jwt`, `bcrypt`, `golang.org/x/oauth2` |
| Events        | Go channels + worker goroutine (`internal/email`) |
| File storage  | Cloudinary Go SDK (streamed, no temp files) |

