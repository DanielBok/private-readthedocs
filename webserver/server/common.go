package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	db "private-sphinx-docs/services/database"
)

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

func Forbid(w http.ResponseWriter, _ *http.Request) { http.Error(w, "forbidden", http.StatusForbidden) }

func Ok(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "okay")
}

func BadRequest(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
	return
}

func authenticate(store IStore, r *http.Request) (*db.Account, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, errors.New("authentication not set in request")
	}

	account, err := store.FetchAccount(username)
	if err != nil {
		return nil, errors.Wrap(err, "server error: could not fetch account")
	}
	if !account.HasValidPassword(password) {
		return nil, errors.New("invalid credentials")
	}

	return account, nil
}
