package database_test

import (
	"testing"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	. "private-sphinx-docs/services/database"
)

const (
	testDoc1      = "Document1"
	adminUsername = "admin"
)

func TestNewAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	for _, r := range []struct {
		Username string
		Password string
		HasError bool
	}{
		{"username", "password", false},
		{"u", "password", true},
		{"username", "p", true},
	} {
		acc, err := NewAccount(r.Username, r.Password, false)
		if r.HasError {
			assert.Error(err)
		} else {
			assert.NoError(err)
			assert.IsType(&Account{}, acc)
		}
	}
}

func TestDatabase_CreateAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	accounts, err := mockAccounts()
	assert.NoError(err)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info)
		assert.NoError(err)

		for _, acc := range accounts {
			acc, err := db.CreateAccount(acc)
			assert.NoError(err)
			assert.IsType(&Account{}, acc)
		}

		// all these should lead to errors
		newAccounts, err := mockAccounts()
		assert.NoError(err)
		acc := newAccounts[0]
		_, err = db.CreateAccount(acc)
		assert.Error(err, "username already exists")

		// test that validation raises errors
		_, err = db.CreateAccount(&Account{
			Username: "SomeName",
			Password: "",
		})
		assert.Error(err, "password too short, validation should have caught it")
	})
}

func TestDatabase_FetchAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)

		for _, r := range []struct {
			Username string
			HasError bool
		}{
			{adminUsername, false},
			{"user0", true},
		} {
			acc, err := db.FetchAccount(r.Username)
			if r.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.IsType(&Account{}, acc)
			}
		}
	})
}

func TestDatabase_UpdateAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)

		for _, r := range []struct {
			Id       int
			Username string
			HasError bool
		}{
			{1, "Username", false},
			{1, "AA", true},
			{10, "Username", true},
			{0, "Username", true},
		} {
			acc := &Account{
				Id:       r.Id,
				Username: r.Username,
				Password: "Password",
				IsAdmin:  false,
			}
			res, err := db.UpdateAccount(acc)
			if r.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.EqualValues(acc, res)
			}
		}
	})
}

func TestDatabase_DeleteAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)

		for _, r := range []struct {
			Username string
			HasError bool
		}{
			{adminUsername, false},
			{"UserDoesNotExist", true},
		} {
			err := db.DeleteAccount(r.Username)
			if r.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		}
	})
}

// Utilities here
func mockAccounts() ([]*Account, error) {
	var accounts []*Account

	for _, v := range []struct {
		Username string
		Password string
		IsAdmin  bool
	}{
		{adminUsername, "password", true},
		{"user1  ", "password", false},
		{"user2", "password", false},
	} {
		u, err := NewAccount(v.Username, v.Password, v.IsAdmin)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, u)
	}
	return accounts, nil
}

func seedAccounts(db *Database) error {
	accounts, err := mockAccounts()
	if err != nil {
		return err
	}

	for _, acc := range accounts {
		_, err := db.CreateAccount(acc)
		if err != nil {
			return err
		}
	}
	return nil
}
