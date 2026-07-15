package initializers

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectToDB(ctx context.Context) (*pgxpool.Pool, error) {
	DB_USER := os.Getenv("DATABASE_USER")
	DB_PASS := os.Getenv("DATABASE_PASS")
	DB_HOST := os.Getenv("DATABASE_HOST")
	DB_NAME := os.Getenv("DATABASE_NAME")
	DB_PORT := os.Getenv("DATABASE_PORT")
	DB_URL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", DB_USER, DB_PASS, DB_HOST, DB_PORT, DB_NAME)
	pool, err := pgxpool.New(ctx, DB_URL)
	if err != nil {
		cleanErr := strings.NewReplacer("\r", "", "\n", "").Replace(err.Error())
		log.Println("Failed to log in to the db", cleanErr)
		return nil, err
	}
	return pool, nil
}
