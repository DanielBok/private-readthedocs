package database

import (
	"time"

	"github.com/pkg/errors"
)

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

func (d *Database) DeleteDocument(name string) error {
	doc, err := d.FetchDocument(name)
	if err != nil {
		return err
	} else if doc == nil {
		return errors.Errorf("No document with name '%s'", name)
	}

	tx := d.MustBegin()
	defer tx.Close(err)

	_, err = tx.Exec(`DELETE FROM document WHERE id = $1`, doc.Id)

	return nil
}
