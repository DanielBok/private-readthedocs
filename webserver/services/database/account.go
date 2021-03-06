package database

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type Account struct {
	Id       int        `json:"id"`
	Username string     `json:"username"`
	Password string     `json:"password,omitempty"`
	IsAdmin  bool       `json:"isAdmin" db:"is_admin"`
	Projects []*Project `json:"projects"`
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
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) == nil
}

func (u *Account) SaltPassword() error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "could not hash password")
	}
	u.Password = string(hash)
	return nil
}

func (d *Database) FetchAccount(username string) (*Account, error) {
	var err error
	tx := d.MustBegin()
	defer tx.Close(err)

	acc := &Account{}
	err = tx.Get(acc, "select * from ACCOUNT where USERNAME = $1", username)
	if err != nil {
		return nil, err
	}

	return acc, nil
}

func (d *Database) FetchAccounts() ([]*Account, error) {
	var err error
	tx := d.MustBegin()
	defer tx.Close(err)

	var accounts []*Account
	err = tx.Select(&accounts, "select * from ACCOUNT")
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func (d *Database) CreateAccount(username, password string, isAdmin bool) (*Account, error) {
	account := &Account{
		Username: username,
		Password: password,
		IsAdmin:  isAdmin,
	}
	err := account.Validate()
	if err != nil {
		return nil, err
	}

	err = account.SaltPassword()
	if err != nil {
		return nil, err
	}

	tx := d.MustBegin()
	defer tx.Close(err)

	rows, err := tx.NamedQuery(`
insert into ACCOUNT (USERNAME, PASSWORD, IS_ADMIN) 
values (:username, :password, :is_admin)
returning ID
`, *account)
	if err != nil {
		return nil, err
	}
	account.Id = mustGetId(rows)

	return account, nil
}

func (d *Database) UpdateAccount(account *Account) (*Account, error) {
	if account.Id <= 0 {
		return nil, errors.New("account id not given")
	}

	err := account.Validate()
	if err != nil {
		return nil, err
	}

	err = account.SaltPassword()
	if err != nil {
		return nil, err
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
