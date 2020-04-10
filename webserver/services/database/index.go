package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

type Database struct {
	db *sqlx.DB
}

type DbOption struct {
	Host     string
	Port     int
	User     string
	Password string
	DbName   string
	SSLMode  string
}

type Tx struct {
	*sqlx.Tx
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
		db, err := sqlx.Connect("postgres", option.ConnectionString(false))
		if err == nil {
			return &Database{db}, nil
		}
		wait += i
		time.Sleep(time.Duration(wait) * time.Second)
	}

	return nil, errors.Errorf("could not connect to database with '%s'", option.ConnectionString(true))
}

func (d *Database) Migrate() error {
	driver, err := postgres.WithInstance(d.db.DB, &postgres.Config{})
	if err != nil {
		return errors.Wrap(err, "could not create database driver")
	}

	sourceUrl, err := generateMigrationFiles()
	if err != nil {
		return errors.Wrap(err, "could not generate migration files")
	}

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

func (d *Database) MustBegin() Tx {
	tx := d.db.MustBegin()
	return Tx{tx}
}

// NamedExec a named query within a transaction. Any named placeholder parameters
// are replaced with fields from arg.
func (t *Tx) NamedExec(query string, arg interface{}) (int64, error) {
	result, err := t.Tx.NamedExec(query, arg)
	if err != nil {
		return 0, err
	}
	num, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return num, err
}

func (t *Tx) Close(err error) {
	if err != nil {
		_ = t.Tx.Rollback()
		log.Printf("error rolling back transaction: %v", err)
	} else {
		_ = t.Tx.Commit()
	}
}

// Use this after inserting data into the database. The query should have a
// "RETURNING id" at the end
func mustGetId(rows *sqlx.Rows) int {
	var id int
	for rows.Next() {
		err := rows.Scan(&id)
		if err != nil {
			panic(errors.Wrap(err, "could not get id from rows"))
		}
		return id
	}
	panic("could not get any id data")
}
