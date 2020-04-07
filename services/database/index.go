package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

type Database struct {
	db *sql.DB
}

type DbOption struct {
	Host     string
	Port     int
	User     string
	Password string
	DbName   string
	SSLMode  string
}

func (o *DbOption) ConnectionString(mask bool) string {
	var password string
	if mask {
		password = strings.Repeat("*", len(o.Password))
	} else {
		password = o.Password
	}
	if o.SSLMode == "" {
		o.SSLMode = "disable"
	}

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		o.Host, o.Port, o.User, password, o.DbName, o.SSLMode)
}

func New(option *DbOption) (*Database, error) {
	wait := 1
	for i := 1; i < 10; i++ {
		db, err := sql.Open("postgres", option.ConnectionString(false))
		if err == nil {
			return &Database{db}, nil
		}
		wait += i
		time.Sleep(time.Duration(wait) * time.Second)
	}

	return nil, errors.Errorf("could not connect to database with '%s'", option.ConnectionString(true))
}

func (d *Database) Migrate() error {
	driver, err := postgres.WithInstance(d.db, &postgres.Config{})
	if err != nil {
		return errors.Wrap(err, "could not create database driver")
	}

	sourceUrl, err := generateMigrationFiles()
	if err != nil {
		return errors.Wrap(err, "could not generate migration files")
	}
	log.Printf("Source URL: %s", sourceUrl)

	m, err := migrate.NewWithDatabaseInstance(sourceUrl, "postgres", driver)
	if err != nil {
		return errors.Wrap(err, "could not create migration instance")
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, "could not apply migrations")
	}
	return nil
}

func (d *Database) Close() error {
	return d.db.Close()
}
