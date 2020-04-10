package database

import (
	"time"

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

type Document struct {
	Id         int       `json:"id"`
	Name       string    `json:"name"`
	LastUpdate time.Time `json:"lastUpdate" db:"last_update"`
	AccountId  int       `json:"-" db:"account_id"`
}

func (d *Document) Validate() error {
	if len(d.Name) < 2 {
		return errors.New("document name must have 2 or more characters")
	} else if d.AccountId <= 0 {
		return errors.New("document must have valid account Id")
	}
	return nil
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

func (d *Database) DeleteAccount(username string) (*Account, error) {
	account, err := d.FetchAccount(username)
	if err != nil {
		return nil, err
	} else if account == nil {
		return nil, errors.Errorf("no account with username: %s", username)
	}
	_, err = account.FetchDocuments()
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch documents from account. "+
			"This would hinder removal of physical documents from the disk drive later")
	}

	tx := d.MustBegin()
	defer tx.Close(err)

	_, err = tx.Exec("DELETE FROM account WHERE username = $1", username)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func NewDocument(name string, accountId int) (*Document, error) {
	doc := &Document{
		Name:       name,
		LastUpdate: time.Now(),
		AccountId:  accountId,
	}

	if err := doc.Validate(); err != nil {
		return nil, err
	}

	return doc, nil
}

func (d *Database) FetchDocument(name string) (*Document, error) {
	var err error
	tx := d.MustBegin()
	defer tx.Close(err)

	document := &Document{}
	err = tx.Get(document, `SELECT * FROM document WHERE name = $1`, name)
	if err != nil {
		return nil, err
	}

	return document, nil
}

func (d *Database) FetchDocuments(accountId int) ([]*Document, error) {
	var err error
	tx := d.MustBegin()
	defer tx.Close(err)

	query := "SELECT id, name, last_update, account_id FROM document"
	var documents []*Document
	if accountId < 0 {
		return nil, errors.Errorf("Invalid account id '%d'. Use 0 if you want to query everything", accountId)
	} else if accountId == 0 {
		err = tx.Select(&documents, query)
		if err != nil {
			return nil, err
		}
		return documents, nil
	} else {
		err = tx.Select(&documents, query+" WHERE account_id = $1", accountId)
		if err != nil {
			return nil, err
		}
		return documents, nil
	}
}

func (d *Database) CreateDocument(document *Document) (*Document, error) {
	err := document.Validate()
	if err != nil {
		return nil, err
	}

	tx := d.MustBegin()
	defer tx.Close(err)

	document.LastUpdate = time.Now()
	rows, err := tx.NamedQuery(`
INSERT INTO document (name, last_update, account_id) 
VALUES (:name, :last_update, :account_id)
RETURNING id
`, document)
	if err != nil {
		return nil, err
	}

	document.Id = mustGetId(rows)

	return document, nil
}

func (d *Database) UpdateDocument(document *Document) (*Document, error) {
	err := document.Validate()
	if err != nil {
		return nil, err
	}

	tx := d.MustBegin()
	defer tx.Close(err)

	document.LastUpdate = time.Now()
	n, err := tx.NamedExec(`
UPDATE document
SET name = :name,
    last_update = :last_update,
	account_id = :account_id
WHERE id = :id
`, document)
	if err != nil {
		return nil, err
	} else if n == 0 {
		return nil, errors.Errorf("no document with id: %d", document.Id)
	}

	return document, nil
}

func (d *Database) DeleteDocument(name string) (*Document, error) {
	doc, err := d.FetchDocument(name)
	if err != nil {
		return nil, err
	} else if doc == nil {
		return nil, errors.Errorf("No document with name '%s'", name)
	}

	tx := d.MustBegin()
	defer tx.Close(err)

	_, err = tx.Exec(`DELETE FROM document WHERE id = $1`, doc.Id)

	return doc, nil
}
