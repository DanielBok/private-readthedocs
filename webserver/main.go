package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"private-sphinx-docs/server"
	db "private-sphinx-docs/services/database"
	sf "private-sphinx-docs/services/staticfiles"
)

//go:generate python scripts/create_migration_file.py
//go:generate python scripts/create_meta_variables.py

func main() {
	config, err := ReadConfig()
	if err != nil {
		log.Fatal(err)
	}

	fh, err := createFileHandler(config)
	if err != nil {
		log.Fatal(err)
	}

	store, err := db.New(config.DbOption())
	if err != nil {
		log.Fatal(err)
	}

	if config.Database.Migrate {
		err = store.Migrate()
		if err != nil {
			log.Fatal(errors.Wrap(err, "could not migrate database"))
		}
		log.Info("Migrated database to latest version")
	}

	srv, err := server.New(server.Option{
		Version:     version,
		Port:        config.App.Port,
		Store:       store,
		FileHandler: fh,
	})
	if err != nil {
		log.Fatal(err)
	}

	// run application
	done := make(chan struct{})
	go shutdownListener(srv, done)
	runServer(srv, config)
	<-done

}

func createFileHandler(config *Config) (server.IFileHandler, error) {
	switch t := strings.ToLower(config.StaticFile.Type); t {
	case "filesys":
		return sf.NewFileSys(config.StaticFile.Source)
	default:
		return nil, errors.Errorf("Unknown static file handler type: %s", t)
	}
}

func runServer(srv *http.Server, config *Config) {
	if config.HasCert() {
		log.Printf("Running server in HTTPS mode at %s", srv.Addr)
		if err := srv.ListenAndServeTLS(config.TLSFiles()); err != http.ErrServerClosed {
			log.Printf("server error - cause: %v", err)
		}
	} else {
		log.Printf("Running server in HTTP mode at %s", srv.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("server error - cause: %v", err)
		}
	}
}

func shutdownListener(srv *http.Server, ch chan struct{}) {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	sig := <-sigint

	log.Printf("shutting down server. cause: received signal %s", sig.String())

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("error shutting down application server: %v", err)
	}

	close(ch)
}
