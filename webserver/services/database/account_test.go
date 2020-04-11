package database_test

import (
	"testing"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	. "private-sphinx-docs/services/database"
)

const (
	admin = "admin"
	user1 = "user1"
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

func TestAccount_HasValidPassword(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	acc := &Account{
		Username: "Username",
		Password: "Password",
		IsAdmin:  false,
		Projects: nil,
	}

	err := acc.SaltPassword()
	assert.NoError(err)

	for _, r := range []struct {
		Password string
		Expected bool
	}{
		{"Password", true},
		{"Wrong", false},
	} {
		actual := acc.HasValidPassword(r.Password)
		assert.Equal(r.Expected, actual)
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
		defer closeDb(db)

		for _, acc := range accounts {
			acc, err := db.CreateAccount(acc.Username, acc.Password, acc.IsAdmin)
			assert.NoError(err)
			assert.IsType(&Account{}, acc)
		}

		// all these should lead to errors
		newAccounts, err := mockAccounts()
		assert.NoError(err)
		acc := newAccounts[0]
		_, err = db.CreateAccount(acc.Username, acc.Password, acc.IsAdmin)
		assert.Error(err, "username already exists")

		// test that validation raises errors
		_, err = db.CreateAccount("SomeName", "", true)
		assert.Error(err, "password too short, validation should have caught it")
	})
}

func TestDatabase_FetchAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)
		defer closeDb(db)

		for _, r := range []struct {
			Username string
			HasError bool
		}{
			{admin, false},
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
		defer closeDb(db)

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
			res, err := db.UpdateAccount(r.Id, r.Username, "Password", false)
			if r.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.IsType(&Account{}, res)
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
		defer closeDb(db)

		for _, r := range []struct {
			Username string
			HasError bool
		}{
			{admin, false},
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
		{admin, "password", true},
		{user1, "password", false},
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
		_, err := db.CreateAccount(acc.Username, acc.Password, acc.IsAdmin)
		if err != nil {
			return err
		}
	}
	return nil
}
