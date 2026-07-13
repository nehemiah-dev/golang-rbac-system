package users

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	usersdb "github.com/Steve-s-Circle-on-System-Design/golang-rbac-system/internal/users/sqlc"
)

type Repository struct {
	pool    *pgxpool.Pool
	queries *usersdb.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:    pool,
		queries: usersdb.New(pool),
	}
}

// --- WRAPPER METHODS ---

func (r *Repository) Create(ctx context.Context, arg usersdb.CreateUserParams) (usersdb.User, error) {
	return r.queries.CreateUser(ctx, arg)
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (usersdb.User, error) {
	return r.queries.GetUserByEmail(ctx, email)
}

func (r *Repository) GetByID(ctx context.Context, id pgtype.UUID) (usersdb.User, error) {
	return r.queries.GetUserByID(ctx, id)
}

func (r *Repository) Verify(ctx context.Context, id pgtype.UUID) error {
	return r.queries.VerifyUser(ctx, id)
}

func (r *Repository) UpdateRole(ctx context.Context, arg usersdb.UpdateUserRoleParams) error {
	return r.queries.UpdateUserRole(ctx, arg)
}
