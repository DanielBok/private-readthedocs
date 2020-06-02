package dto

import (
	"strings"

	db "private-sphinx-docs/services/database"
)

type Account struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	IsAdmin  bool   `json:"isAdmin,omitempty" db:"is_admin"`
}

type AccountUpdate struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	IsAdmin  bool   `json:"isAdmin,omitempty" db:"is_admin"`
}

// Converts to db.Account
func (a *AccountUpdate) Cast() *db.Account {
	return &db.Account{
		Id:       a.Id,
		Username: strings.TrimSpace(a.Username),
		Password: strings.TrimSpace(a.Password),
		IsAdmin:  a.IsAdmin,
	}
}
