package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	defaultMaxAttempts = 20
	defaultMaxTimeout  = time.Second * time.Duration(2)
)

func init() {
	URL := fmt.Sprintf(
		"postgres://%s:%s@postgres:5432/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	var (
		m   *migrate.Migrate
		err error
	)

	for attempt := defaultMaxAttempts; attempt > 0; attempt-- {
		m, err = migrate.New("file://migrations", URL)
		if err == nil {
			break
		}

		log.Printf("migrations: trying to connect to db, attempts left: %d", attempt)

		time.Sleep(defaultMaxTimeout)
	}

	if err != nil {
		log.Fatalf("migrations: error occured during trying to connect to db: %s", err)
	}

	err = m.Up()
	defer func(m *migrate.Migrate) {
		err, _ := m.Close()
		if err != nil {
			log.Fatalf("migrations: error occured during trying to disconnect from db: %s", err)
		}
	}(m)

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migrations: error occured during trying to run up migrations: %s", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("migrations: no new change")
	}

	log.Println("migrations: success")
}
