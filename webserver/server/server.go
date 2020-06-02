package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type subdomain string

const (
	main subdomain = "main"
	docs subdomain = "docs"
)

type Option struct {
	Version     string
	Port        int
	Store       IStore
	FileHandler IFileHandler
}

type SubDomains map[subdomain]http.Handler

func (s SubDomains) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(r.Host, ".")
	switch len(urlParts) {
	case 1:
		s[main].ServeHTTP(w, r)
	case 2:
		s[docs].ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

func New(option Option) (*http.Server, error) {
	subdomains := make(SubDomains)
	subdomains[main] = apiRouter(option)
	subdomains[docs] = docRouter(option)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", option.Port),
		Handler: subdomains,
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

func apiRouter(option Option) *chi.Mux {
	r := chi.NewRouter()
	store := option.Store
	fs := option.FileHandler

	attachMiddleware(r)
	r.Get("/__status", StatusCheck(option.Version))

	r.Route("/api", func(r chi.Router) {
		r.Route("/account", func(r chi.Router) {
			handler := AccountHandler{DB: store, FS: fs}

			r.Get("/", handler.ValidateAccount())
			r.Post("/", handler.CreateAccount())
			r.Put("/", handler.UpdateAccount())
			r.Delete("/{username}", handler.DeleteAccount())
		})

		r.Route("/project", func(r chi.Router) {
			handler := ProjectHandler{DB: store, FS: fs}
			r.Get("/", handler.FetchProjects())           // get all projects
			r.Get("/{username}", handler.FetchProjects()) // get all user projects
			r.Post("/", handler.UploadProject())          // upload new project (create / update)
			r.Delete("/{title}", handler.DeleteProject()) // removes project
		})
	})

	return r
}

func docRouter(option Option) *chi.Mux {
	r := chi.NewRouter()
	attachMiddleware(r)

	handler := DocumentationHandler{option.FileHandler.Source()}
	handler.MustInit()
	r.Handle("/*", handler.FileServer())

	return r
}

func StatusCheck(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		toJson(w, struct {
			Status  string `json:"status"`
			Version string `json:"version"`
		}{"Okay", version})
	}
}
