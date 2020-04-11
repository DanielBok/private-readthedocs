package server

import (
	"net/http"
	"strings"

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

type CreateAccountPayload struct {
	Requester *Credentials `json:"requester"`
	*db.Account
}

type UpdateAccountPayload struct {
	Requester *Credentials `json:"requester"`
	Details   *struct {
		*Credentials
		NewUsername string `json:"new_username"`
		IsAdmin     bool   `json:"isAdmin"`
	} `json:"details"`
}

type DeleteAccountPayload struct {
	Requester *Credentials `json:"requester"`
	Details   *Credentials `json:"details"`
}

func (h *AccountHandler) CreateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p *CreateAccountPayload
		err := readJson(r, &p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		isAdmin := false
		if p.Requester != nil {
			req, err := h.fetchAccount(p.Requester.Username, p.Requester.Password)
			if err == nil && req.IsAdmin {
				// only allow isAdmin to potentially be true if requester is admin
				isAdmin = p.IsAdmin
			}
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

		toJson(w, account)
	}
}

func (h *AccountHandler) UpdateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p *UpdateAccountPayload
		err := readJson(r, &p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		acc, reqIsAdmin, err := h.isAuthorizedToTransformSubject(p.Requester, p.Details.Credentials)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if username := strings.TrimSpace(p.Details.NewUsername); username != "" {
			acc.Username = username
		}
		if reqIsAdmin {
			acc.IsAdmin = p.Details.IsAdmin
		}

		account, err := h.DB.UpdateAccount(acc.Id, acc.Username, acc.Password, acc.IsAdmin)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		toJson(w, account)
	}
}

func (h *AccountHandler) DeleteAccount() http.HandlerFunc {
	removeProjectFiles := func(account *db.Account) error {
		docs, err := account.FetchProjects()
		if err != nil {
			return errors.Wrap(err, "could not get account's projects")
		}
		for _, d := range docs {
			if e := h.FS.Remove(d.Title); e != nil {
				err = multierror.Append(err, errors.Wrapf(err, "could not remove project '%s;", d.Title))
			}
		}
		return err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var p *DeleteAccountPayload
		err := readJson(r, &p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		account, _, err := h.isAuthorizedToTransformSubject(p.Requester, p.Details)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		err = h.DB.DeleteAccount(account.Username)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		err = removeProjectFiles(account)
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

		_, err = h.fetchAccount(p.Username, p.Password)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		ok(w)
	}
}

func (h *AccountHandler) fetchAccount(username, password string) (*db.Account, error) {
	acc, err := h.DB.FetchAccount(username)
	if err != nil {
		return nil, err
	}
	if !acc.HasValidPassword(password) {
		return nil, errors.New("invalid credentials")
	}

	return acc, nil
}

func (h *AccountHandler) isAuthorizedToTransformSubject(requester *Credentials, subject *Credentials) (*db.Account, bool, error) {
	req, err := h.fetchAccount(requester.Username, requester.Password)
	if err != nil {
		return nil, false, err
	} else if !req.IsAdmin && subject.Username != req.Username {
		return nil, false, errors.New("Unauthorized to make changes")
	}

	account, err := h.DB.FetchAccount(subject.Username)
	if err != nil {
		return nil, false, err
	}
	return account, req.IsAdmin, nil
}
