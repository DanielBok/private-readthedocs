package database

import (
	"github.com/pkg/errors"
)

type Account struct {
	Id        int         `json:"id"`
	Username  string      `json:"username"`
	Password  string      `json:"password,omitempty"`
	IsAdmin   bool        `json:"isAdmin" db:"is_admin"`
	Documents []*Document `json:"documents"`
	db        *Database   `json:"-" db:"-"`
}

func NewAccount(username, password string, isAdmin bool) (*Account, error) {
	u := &Account{
		Username: username,
		Password: password,
		IsAdmin:  isAdmin,
	}

	if err := u.Validate(); err != nil {
		return nil, err
	}

	return u, nil
}

func (u *Account) Validate() error {
	if len(u.Username) < 4 {
		return errors.New("username must have 4 characters or more")
	}

	if len(u.Password) < 4 {
		return errors.New("password must have 4 characters or more")
	}

	return nil
}

func (u *Account) HasValidPassword(password string) bool {
	return u.Password == password
}

func (u *Account) FetchDocuments() ([]*Document, error) {
	docs, err := u.db.FetchDocuments(u.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not fetch documents from %s", u.Username)
	}
	u.Documents = docs
	return docs, nil
}

func (d *Database) FetchAccount(username string) (*Account, error) {
	var err error
	tx := d.MustBegin()
	defer tx.Close(err)

	acc := &Account{}
	err = tx.Get(acc, "SELECT * FROM account WHERE username = $1", username)
	if err != nil {
		return nil, err
	}
	acc.db = d

	return acc, nil
}

func (d *Database) CreateAccount(account *Account) (*Account, error) {
	err := account.Validate()
	if err != nil {
		return nil, err
	}

	tx := d.MustBegin()
	defer tx.Close(err)

	rows, err := tx.NamedQuery(`
INSERT INTO account (username, password, is_admin) 
VALUES (:username, :password, :is_admin)
RETURNING id
`, *account)
	if err != nil {
		return nil, err
	}
	account.Id = mustGetId(rows)
	account.db = d

	return account, nil
}

func (d *Database) UpdateAccount(account *Account) (*Account, error) {
	err := account.Validate()
	if err != nil {
		return nil, err
	}
	if account.Id <= 0 {
		return nil, errors.New("account id not given")
	}
	tx := d.MustBegin()
	defer tx.Close(err)

	n, err := tx.NamedExec(`
UPDATE account
SET username = :username,
    password = :password,
    is_admin = :is_admin
WHERE id = :id;
`, account)
	if err != nil {
		return nil, err
	} else if n == 0 {
		return nil, errors.Errorf("no account with id: %d", account.Id)
	}

	account.db = d
	return account, nil
}

func (d *Database) DeleteAccount(username string) error {
	var err error
	tx := d.MustBegin()
	defer tx.Close(err)

	n, err := tx.Exec("DELETE FROM account WHERE username = $1", username)
	if err != nil {
		return err
	} else if n == 0 {
		return errors.Errorf("no account with username: '%s'", username)
	}

	return nil
}
