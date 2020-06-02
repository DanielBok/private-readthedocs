package server

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"private-sphinx-docs/server/dto"
	db "private-sphinx-docs/services/database"
)

type AccountHandler struct {
	DB IStore
	FS IFileHandler
}

func (h *AccountHandler) CreateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p *dto.Account
		err := readJson(r, &p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		isAdmin := false
		// get requester, if there's an error, it just means that requester is not admin user
		req, err := authenticate(h.DB, r)
		if err == nil && req != nil && req.IsAdmin {
			// only allow admin to set admin
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
	return func(w http.ResponseWriter, r *http.Request) {
		account, err := authenticate(h.DB, r)
		if err != nil {
			Forbid(w, r)
			return
		}

		var p *dto.AccountUpdate
		err = readJson(r, &p)
		if err != nil {
			BadRequest(w, err)
			return
		}

		// check that user can change account. If requester is admin, can change everything.
		// Otherwise, ensure that the requester is changing the same account (by id)
		if !(account.IsAdmin || account.Id == p.Id) {
			Forbid(w, r)
			return
		}
		if !account.IsAdmin {
			p.IsAdmin = false
		}

		account, err = h.DB.UpdateAccount(p.Cast())
		if err != nil {
			BadRequest(w, err)
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
		account, err := authenticate(h.DB, r)
		if err != nil {
			Forbid(w, r)
			return
		}

		username := chi.URLParam(r, "username")
		if !(account.IsAdmin || account.Username == username) {
			Forbid(w, r)
			return
		}

		// all validation done, now we get the account
		account, err = h.DB.FetchAccount(username)
		if err != nil {
			BadRequest(w, err)
			return
		}

		projects, err := h.DB.FetchProjectsByAccount(account.Id)
		if err != nil {
			BadRequest(w, err)
			return
		}

		err = removeProjectFiles(projects)
		if err != nil {
			BadRequest(w, err)
			return
		}

		err = h.DB.DeleteAccount(username)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		Ok(w, r)
	}
}

func (h *AccountHandler) ValidateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := authenticate(h.DB, r)
		if err != nil {
			Forbid(w, r)
			return
		}

		Ok(w, r)
	}
}
