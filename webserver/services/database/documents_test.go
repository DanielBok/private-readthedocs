package database_test

import (
	"testing"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	. "private-sphinx-docs/services/database"
)

func TestNewDocument(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	for _, r := range []struct {
		Name      string
		AccountId int
		HasError  bool
	}{
		{"package", 1, false},
		{"package", 0, true},
		{"p", 1, true},
	} {
		doc, err := NewDocument(r.Name, r.AccountId)
		if r.HasError {
			assert.Error(err)
		} else {
			assert.NoError(err)
			assert.IsType(&Document{}, doc)
		}
	}
}

func TestDatabase_CreateDocument(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	documents, err := mockDocuments()
	assert.NoError(err)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)
		acc, err := db.FetchAccount(adminUsername)
		assert.NoError(err)

		for _, d := range documents {
			d.AccountId = acc.Id
			doc, err := db.CreateDocument(d)
			assert.NoError(err)
			assert.IsType(&Document{}, doc)
		}

		_, err = db.CreateDocument(documents[0])
		assert.Error(err, "document already exists")

		// test that validation raises errors
		_, err = db.CreateDocument(&Document{
			Name:      "",
			AccountId: acc.Id,
		})
		assert.Error(err, "validation failed")
	})
}

func TestDatabase_FetchDocument(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedDocuments)
		assert.NoError(err)

		for _, r := range []struct {
			Name     string
			HasError bool
		}{
			{testDoc1, false},
			{"DoesNotExist", true},
		} {
			doc, err := db.FetchDocument(r.Name)
			if r.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.IsType(&Document{}, doc)
			}
		}
	})
}

func TestDatabase_FetchDocuments(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedDocuments)
		assert.NoError(err)

		acc, err := db.FetchAccount(adminUsername)
		assert.NoError(err)

		docs, err := db.FetchDocuments(acc.Id)
		assert.NoError(err)
		assert.Greater(len(docs), 0)
	})
}

func TestDatabase_UpdateDocument(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedDocuments)
		assert.NoError(err)

		doc, err := db.FetchDocument(testDoc1)
		assert.NoError(err)

		_, err = db.UpdateDocument(&Document{
			Id:        doc.Id,
			Name:      "NewName",
			AccountId: doc.AccountId,
		})
		assert.NoError(err)
	})
}

func TestDatabase_DeleteDocument(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedDocuments)
		assert.NoError(err)

		for _, r := range []struct {
			Name     string
			HasError bool
		}{
			{testDoc1, false},
			{"DoesNotExist", true},
		} {
			err := db.DeleteDocument(r.Name)
			if r.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		}
	})
}

// Utilities here
func mockDocuments() ([]*Document, error) {
	var documents []*Document

	for _, v := range []struct {
		Name      string
		AccountId int
	}{
		{testDoc1, 1},
		{"Document2", 1},
	} {
		d, err := NewDocument(v.Name, v.AccountId)
		if err != nil {
			return nil, err
		}
		documents = append(documents, d)
	}
	return documents, nil
}

func seedDocuments(db *Database) error {
	documents, err := mockDocuments()
	if err != nil {
		return err
	}

	acc, err := db.FetchAccount(adminUsername)
	if err != nil {
		return err
	}

	for _, d := range documents {
		d.AccountId = acc.Id
		_, err = db.CreateDocument(d)
		if err != nil {
			return err
		}
	}
	return nil
}
