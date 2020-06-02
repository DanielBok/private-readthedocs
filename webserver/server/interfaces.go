package server

import (
	"io"

	db "private-sphinx-docs/services/database"
)

type IStore interface {
	FetchAccount(username string) (*db.Account, error)
	FetchAccounts() ([]*db.Account, error)
	CreateAccount(username, password string, isAdmin bool) (*db.Account, error)
	UpdateAccount(account *db.Account) (*db.Account, error)
	DeleteAccount(username string) (*db.Account, error)

	FetchProjects() ([]*db.Project, error)
	FetchProjectsByAccount(accountId int) ([]*db.Project, error)
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
	Source() string
}
