package db

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Коннект и создание конфига.
func (db *DB) Init() {
	db.name = "Postgres"
	var err error
	dbUrl := fmt.Sprintf("postgres://%s:%s@%s/%s", os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_NAME"))

	cfg, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		log.Fatalf("%v: Init() error: %s\n", db.name, err)
	}

	db.pool, err = pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("%v: can't connect to DB: %v\n", db.name, err)
	}
	log.Printf("%v: SUCCESS\n", db.name)
}
