package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	db "private-sphinx-docs/services/database"
)

type IStore interface {
	FetchAccount(username string) (*db.Account, error)
	FetchAccounts() ([]*db.Account, error)
	CreateAccount(username, password string, isAdmin bool) (*db.Account, error)
	UpdateAccount(id int, username, password string, isAdmin bool) (*db.Account, error)
	DeleteAccount(username string) error

	FetchProjects() ([]*db.Project, error)
	CreateOrUpdateProject(accountId int, title string) (*db.Project, error)
	DeleteProject(title string) error
	CanOwnProject(accountId int, title string) (bool, error)
}

type IFileHandler interface {
	// Decompresses the uplaoded zip file and saves it
	Upload(r io.ReaderAt, name string, size int64) error
	// Gets the destination path for the static files
	Destination(name string) string
	// Remove the project files
	Remove(name string) error
}

type Option struct {
	Version     string
	Port        int
	Store       IStore
	FileHandler IFileHandler
}

func New(option Option) (*http.Server, error) {
	r := chi.NewRouter()
	attachMiddleware(r)
	attachHandlers(r, option)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", option.Port),
		Handler: r,
	}, nil
}

func attachMiddleware(r *chi.Mux) {
	r.Use(middleware.RequestID,
		middleware.Compress(5),
		middleware.Recoverer,
		middleware.RealIP,
		middleware.Logger,
	)
}

func attachHandlers(r *chi.Mux, option Option) {
	store := option.Store
	fs := option.FileHandler

	r.Get("/__status", StatusCheck(option.Version))

	r.Route("/api", func(r chi.Router) {
		r.Route("/account", func(r chi.Router) {
			handler := AccountHandler{DB: store, FS: fs}

			r.Get("/validate", handler.ValidateAccount())
			r.Post("/", handler.CreateAccount())
			r.Put("/", handler.UpdateAccount())
			r.Delete("/", handler.DeleteAccount())
		})

		r.Route("/project", func(r chi.Router) {
			handler := ProjectHandler{DB: store, FS: fs}
			r.Get("/", handler.FetchProjects())           // get all projects
			r.Get("/{username}", handler.FetchProjects()) // get all user projects
			r.Post("/", handler.UploadProject())          // upload new project (create / update)
			r.Delete("/", handler.DeleteProject())        // removes project
		})
	})

}

func StatusCheck(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		toJson(w, struct {
			Status  string `json:"status"`
			Version string `json:"version"`
		}{"Okay", version})
	}
}

func toJson(w http.ResponseWriter, object interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(object); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func readJson(r *http.Request, object interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(object); err != nil {
		return err
	}
	return nil
}

func ok(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, "Okay"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
