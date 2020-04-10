package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type IStore interface {
	IAccountStore
}

type Option struct {
	Version string
	Port    int
	Store   IStore
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
	r.Get("/__status", StatusCheck(option.Version))

	r.Route("/api", func(r chi.Router) {
		r.Route("/account", func(r chi.Router) {
			handler := AccountHandler{DB: store}

			r.Get("/validate", handler.ValidateAccount())
			r.Post("/", handler.CreateAccount())
			r.Put("/", handler.UpdateAccount())
			r.Delete("/", handler.DeleteAccount())
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
