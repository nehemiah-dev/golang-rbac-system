package initializers

import (
	"log"
	"strconv"

	"github.com/joho/godotenv"
)

func LoadConfig() error {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Failed to load config from env because", err.Error())
	}
	return nil
}

// IsValidPort reports whether p is a syntactically valid TCP port number (1-65535).
func IsValidPort(p string) bool {
	if p == "" || len(p) > 5 {
		return false
	}
	for _, r := range p {
		if r < '0' || r > '9' {
			return false
		}
	}
	n, err := strconv.Atoi(p)
	return err == nil && n > 0 && n <= 65535
}
