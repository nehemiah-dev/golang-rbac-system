package initializers

import (
	"log"

	"github.com/joho/godotenv"
)

func LoadConfig() error {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Failed to load config from env because", err.Error())
	}
	return nil
}
