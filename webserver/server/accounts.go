package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	db "private-sphinx-docs/services/database"
)

type AccountHandler struct {
	DB IStore
	FS IFileHandler
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UpdateAccountPayload struct {
	*Credentials
	OldUsername string `json:"oldUsername"`
	IsAdmin     bool   `json:"isAdmin"`
}

func (h *AccountHandler) CreateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p *db.Account
		err := readJson(r, &p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		isAdmin := false
		// get requester, if there's an error, it just means that requester is not
		// admin user
		req, err := authenticate(h.DB, r)
		if err == nil && req != nil && req.IsAdmin {
			// only allow isAdmin to potentially be true if requester is admin
			isAdmin = p.IsAdmin
		}

		if accounts, err := h.DB.FetchAccounts(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		} else if len(accounts) == 0 {
			isAdmin = true // first account is always admin account
		}

		account, err := h.DB.CreateAccount(p.Username, p.Password, isAdmin)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		// mask password
		account.Password = ""

		toJson(w, account)
	}
}

func (h *AccountHandler) UpdateAccount() http.HandlerFunc {
	formAccount := func(req *db.Account, p *UpdateAccountPayload) (*db.Account, error) {
		account, err := h.DB.FetchAccount(p.OldUsername)
		if err != nil {
			return nil, err
		}

		if !(req.IsAdmin || req.Username == account.Username) {
			return nil, errors.New("Unauthorized to make changes")
		}

		account.Username = strings.TrimSpace(p.Username)
		account.Password = strings.TrimSpace(p.Password)

		if req.IsAdmin {
			account.IsAdmin = p.IsAdmin
		}

		return account, nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var p *UpdateAccountPayload
		err := readJson(r, &p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		req, err := authenticate(h.DB, r)
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid requester").Error(), 400)
			return
		}

		acc, err := formAccount(req, p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		account, err := h.DB.UpdateAccount(acc)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		account.Password = ""

		toJson(w, account)
	}
}

func (h *AccountHandler) DeleteAccount() http.HandlerFunc {
	removeProjectFiles := func(projects []*db.Project) error {
		var err error
		for _, d := range projects {
			if e := h.FS.Remove(d.Title); e != nil {
				err = multierror.Append(err, errors.Wrapf(err, "could not remove project '%s;", d.Title))
			}
		}
		return err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		req, err := authenticate(h.DB, r)
		if err != nil {
			http.Error(w, errors.Wrap(err, "invalid requester").Error(), 400)
			return
		}

		username := chi.URLParam(r, "username")
		if !(req.IsAdmin || req.Username == username) {
			http.Error(w, "invalid credentials to remove account", 400)
			return
		}

		account, err := h.DB.DeleteAccount(username)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		err = removeProjectFiles(account.Projects)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		ok(w)
	}
}

func (h *AccountHandler) ValidateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p *Credentials
		err := readJson(r, &p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		acc, err := h.DB.FetchAccount(p.Username)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if !acc.HasValidPassword(p.Password) {
			http.Error(w, "invalid credentials", 400)
			return
		}

		ok(w)
	}
}
