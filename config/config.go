package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	HTTPPort         int
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	ServiceUser      string
	ServicePassword  string
	ClientID         string
	ClientSecret     string
	IP               string
	AuthToken        string
	TargetID         string
}

func NewConfigFromEnv() Config {
	httpPort, err := strconv.Atoi(os.Getenv("HTTP_PORT"))
	if err != nil {
		log.Fatalln(err)
	}

	return Config{
		HTTPPort:         httpPort,
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),
		ServiceUser:      os.Getenv("SERVICE_USER"),
		ServicePassword:  os.Getenv("SERVICE_PASSWORD"),
		ClientID:         os.Getenv("CLIENT_ID"),
		ClientSecret:     os.Getenv("CLIENT_SECRET"),
		IP:               os.Getenv("IP"),
		AuthToken:        os.Getenv("AUTH_TOKEN"),
		TargetID:         os.Getenv("TARGET_ID"),
	}
}
