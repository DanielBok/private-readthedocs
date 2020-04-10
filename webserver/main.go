package main

import (
	"log"

	"private-sphinx-docs/services/database"
)

//go:generate go run services/database/migrations/generate.go
//go:generate python scripts/create_meta_variables.py

func main() {
	db, err := database.New(&database.DbOption{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "password",
		DbName:   "postgres",
		SSLMode:  "disable",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	err = db.Migrate()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Connected")
}
