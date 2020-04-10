package server

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"

	db "private-sphinx-docs/services/database"
)

type IAccountStore interface {
	FetchAccount(username string) (*db.Account, error)
	CreateAccount(account *db.Account) (*db.Account, error)
	UpdateAccount(account *db.Account) (*db.Account, error)
	DeleteAccount(username string) error
}

type AccountHandler struct {
	DB IAccountStore
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
		var account *db.Account
		err := readJson(r, &account)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		account, err = h.DB.CreateAccount(account)
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

		account, isAdmin, err := h.IsAuthorizedToTransformSubject(p.Requester, p.Details.Credentials)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if username := strings.TrimSpace(p.Details.NewUsername); username != "" {
			account.Username = username
		}
		if isAdmin {
			account.IsAdmin = p.Details.IsAdmin
		}

		account, err = h.DB.UpdateAccount(account)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		toJson(w, account)
	}
}

func (h *AccountHandler) DeleteAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p *DeleteAccountPayload
		err := readJson(r, &p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		account, _, err := h.IsAuthorizedToTransformSubject(p.Requester, p.Details)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// TODO remove physical documents
		_, err = account.FetchDocuments()
		if err != nil {
			http.Error(w, errors.Wrap(err, "could not get account's documents").Error(), 400)
			return
		}

		err = h.DB.DeleteAccount(account.Username)
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
		} else if !acc.HasValidPassword(p.Password) {
			http.Error(w, "invalid credentials", 400)
			return
		}

		ok(w)
	}
}

func (h *AccountHandler) IsAuthorizedToTransformSubject(requester *Credentials, subject *Credentials) (*db.Account, bool, error) {
	req, err := h.DB.FetchAccount(requester.Username)
	if err != nil {
		return nil, false, err
	} else if !req.HasValidPassword(requester.Password) || (!req.IsAdmin && subject.Username != req.Username) {
		return nil, false, errors.New("Unauthorized to make changes")
	}

	account, err := h.DB.FetchAccount(subject.Username)
	if err != nil {
		return nil, false, err
	}
	return account, req.IsAdmin, nil
}
