package database_test

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/dhui/dktest"
	"github.com/pkg/errors"

	. "private-sphinx-docs/services/database"
)

var (
	user                 = "postgres"
	password             = "password"
	dbName               = "postgres"
	imageName            = "postgres:12-alpine"
	postgresImageOptions = dktest.Options{
		ReadyFunc:    dbReady,
		PortRequired: true,
		ReadyTimeout: 5 * time.Minute,
		Env: map[string]string{
			"POSTGRES_USER":     user,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       dbName,
		},
	}
)

func getDbOption(c dktest.ContainerInfo) (*DbOption, error) {
	ip, strPort, err := c.FirstPort()
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(strPort)
	return &DbOption{
		Host:     ip,
		Port:     port,
		User:     user,
		Password: password,
		DbName:   dbName,
		SSLMode:  "disable",
	}, nil
}

func dbReady(ctx context.Context, c dktest.ContainerInfo) bool {
	option, err := getDbOption(c)
	if err != nil {
		return false
	}

	db, err := sql.Open("postgres", option.ConnectionString(false))
	if err != nil {
		return false
	}
	defer func() { _ = db.Close() }()

	return db.PingContext(ctx) == nil
}

func newTestDb(c dktest.ContainerInfo, seedFns ...func(d *Database) error) (*Database, error) {
	option, err := getDbOption(c)
	if err != nil {
		return nil, errors.Wrap(err, "could not obtain test postgres db network address")
	}

	store, err := New(option)
	if err != nil {
		return nil, errors.Wrap(err, "could not create test postgres db")
	}

	if err = store.Migrate(); err != nil {
		return nil, err
	}

	for _, f := range seedFns {
		if err := f(store); err != nil {
			return nil, err
		}
	}

	return store, nil
}
