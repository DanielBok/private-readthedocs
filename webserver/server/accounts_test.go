package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	. "private-sphinx-docs/server"
	db "private-sphinx-docs/services/database"
)

func NewAccountHandler() *AccountHandler {
	return &AccountHandler{
		DB: NewMockStore(),
		FS: NewFileHandler(),
	}
}

func TestAccountHandler_CreateAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewAccountHandler()

	for _, s := range []struct {
		Username   string
		Password   string
		IsAdmin    bool
		UseAdmin   bool
		StatusCode int
	}{
		{"User2", "password", false, false, http.StatusOK},
		{"User3", "password", true, false, http.StatusOK},
		{"User4", "password", true, true, http.StatusOK},
		{"User5", "", true, false, http.StatusBadRequest},
	} {
		w := httptest.NewRecorder()
		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(&db.Account{
			Username: s.Username,
			Password: s.Password,
			IsAdmin:  s.IsAdmin,
		})
		assert.NoError(err)
		r := NewTestRequest("POST", "/", &buf, nil)
		if s.UseAdmin {
			r.SetBasicAuth("admin", "password")
		}

		handler.CreateAccount()(w, r)
		assert.Equal(s.StatusCode, w.Code)

		if s.StatusCode == http.StatusOK {
			resp := w.Result()
			var result *db.Account
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(err)
			assert.Empty(result.Password)
			assert.Equal(s.Username, result.Username)
			if s.UseAdmin {
				assert.Equal(s.IsAdmin, result.IsAdmin)
			} else {
				assert.False(result.IsAdmin)
			}

			assert.NoError(resp.Body.Close())
		}
	}
}

func TestAccountHandler_UpdateAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	for _, s := range []struct {
		UseAdmin    bool
		OldUsername string
		IsAdmin     bool
		Password    string
		StatusCode  int
	}{
		{false, "user1", false, "password", http.StatusOK},
		{false, "user1", true, "password", http.StatusOK},
		{true, "user1", true, "password", http.StatusOK},
		{true, "user1", false, "password", http.StatusOK},
		{false, "user1", false, "badPwd", http.StatusBadRequest},
		{false, "badUid", false, "password", http.StatusBadRequest},
	} {
		handler := NewAccountHandler()
		user1, err := handler.DB.CreateAccount("user1", "password", false)
		assert.NoError(err)

		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(&UpdateAccountPayload{
			Credentials: &Credentials{
				Username: "NewUsername",
				Password: "NewPassword",
			},
			OldUsername: s.OldUsername,
			IsAdmin:     s.IsAdmin,
		})
		assert.NoError(err)

		r := NewTestRequest("PUT", "/", &buf, nil)
		if s.UseAdmin {
			r.SetBasicAuth("admin", "password")
		} else {
			r.SetBasicAuth(user1.Username, s.Password)
		}

		w := httptest.NewRecorder()
		handler.UpdateAccount()(w, r)
		assert.Equal(s.StatusCode, w.Code)
		assert.NoError(err)

		if s.StatusCode == http.StatusOK {
			resp := w.Result()
			var result *db.Account
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(err)
			assert.Empty(result.Password)
			assert.Equal("NewUsername", result.Username)
			if s.UseAdmin {
				assert.Equal(s.IsAdmin, result.IsAdmin)
			} else {
				assert.False(result.IsAdmin)
			}

			assert.NoError(resp.Body.Close())
		}
	}
}

func TestAccountHandler_DeleteAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	for _, s := range []struct {
		UseAdmin   bool
		Uid        string
		Pwd        string
		StatusCode int
	}{
		{false, "user1", "password", http.StatusOK},
		{true, "user1", "password", http.StatusOK},
		{true, "user1", "badPwd", http.StatusOK},
		{false, "user1", "badPwd", http.StatusBadRequest},
		{false, "user2", "password", http.StatusBadRequest},
	} {
		handler := NewAccountHandler()
		_, err := handler.DB.CreateAccount("user1", "password", false)
		assert.NoError(err)

		r := NewTestRequest("DELETE", "/", nil, map[string]string{
			"username": s.Uid,
		})
		if s.UseAdmin {
			r.SetBasicAuth("admin", "password")
		} else {
			r.SetBasicAuth(s.Uid, s.Pwd)
		}

		w := httptest.NewRecorder()
		handler.DeleteAccount()(w, r)
		assert.Equal(s.StatusCode, w.Code)
		assert.NoError(err)
	}
}

func TestAccountHandler_ValidateAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	handler := NewAccountHandler()
	_, err := handler.DB.CreateAccount("user1", "password", false)
	assert.NoError(err)

	for _, s := range []struct {
		Uid        string
		Pwd        string
		StatusCode int
	}{
		{"user1", "password", http.StatusOK},
		{"user1", "badPwd", http.StatusBadRequest},
		{"user2", "password", http.StatusBadRequest},
	} {
		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(&Credentials{
			Username: s.Uid,
			Password: s.Pwd,
		})
		assert.NoError(err)

		r := NewTestRequest("POST", "/validate", &buf, nil)
		w := httptest.NewRecorder()
		handler.ValidateAccount()(w, r)
		assert.Equal(s.StatusCode, w.Code)
	}
}
