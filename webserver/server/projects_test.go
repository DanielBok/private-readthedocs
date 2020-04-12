package server_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	. "private-sphinx-docs/server"
	db "private-sphinx-docs/services/database"
)

func NewProjectHandler() *ProjectHandler {
	return &ProjectHandler{
		DB: NewMockStore(),
		FS: NewFileHandler(),
	}
}

func TestProjectHandler_FetchProjects(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewProjectHandler()

	user, err := handler.DB.CreateAccount("user1", "password", false)
	assert.NoError(err)
	_, err = handler.DB.CreateOrUpdateProject(user.Id, "NewProject")
	assert.NoError(err)

	for _, s := range []struct {
		Username   string
		Count      int
		StatusCode int
	}{
		{"", 2, http.StatusOK},
		{"user1", 1, http.StatusOK},
		{"user2", 0, http.StatusBadRequest},
	} {
		r := NewTestRequest("DELETE", "/", nil, map[string]string{
			"username": s.Username,
		})

		w := httptest.NewRecorder()
		handler.FetchProjects()(w, r)
		assert.Equal(s.StatusCode, w.Code)

		if s.StatusCode == http.StatusOK {
			resp := w.Result()
			var projects []*db.Project
			err = json.NewDecoder(resp.Body).Decode(&projects)
			assert.NoError(err)
			assert.Len(projects, s.Count)
			assert.NoError(resp.Body.Close())
		}
	}
}

func TestProjectHandler_UploadProject(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewProjectHandler()

	user, err := handler.DB.CreateAccount("user1", "password", false)
	assert.NoError(err)
	_, err = handler.DB.CreateOrUpdateProject(user.Id, "NewProject")
	assert.NoError(err)

	for _, s := range []struct {
		Username   string
		Password   string
		Title      string
		StatusCode int
	}{
		{"user1", "password", "NewProject", http.StatusOK},
		{"user1", "password", "NewProject2", http.StatusOK},
		{"user1", "badPwd", "NewProject", http.StatusBadRequest},
	} {
		// setup
		body, contentType, err := createUploadPackagePayload(s.Title)
		assert.NoError(err)

		r := NewTestRequest("POST", "/", body, nil)
		r.Header.Set("Content-Type", contentType)
		r.SetBasicAuth(s.Username, s.Password)
		w := httptest.NewRecorder()

		handler.UploadProject()(w, r)
		assert.Equal(s.StatusCode, w.Code)

		if s.StatusCode == http.StatusOK {
			resp := w.Result()
			var project *db.Project
			err = json.NewDecoder(resp.Body).Decode(&project)
			assert.NoError(err)
			assert.NoError(resp.Body.Close())
			assert.IsType(&db.Project{}, project)
		}
	}
}

func createUploadPackagePayload(title string) (io.ReadWriter, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer func() { _ = writer.Close() }()

	if err := writer.WriteField("title", title); err != nil {
		return nil, "", err
	}

	parts, err := writer.CreateFormFile("content", "any-name.zip")
	if err != nil {
		return nil, "", err
	}

	// copy random file content into parts (form file)
	if _, err := io.Copy(parts, bufio.NewReader(bytes.NewBufferString("Random Content"))); err != nil {
		return nil, "", err
	}

	return body, writer.FormDataContentType(), nil
}

func TestProjectHandler_DeleteProject(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	user1 := "user1"
	password := "password"
	title := "project2"

	for _, s := range []struct {
		Username   string
		Password   string
		Title      string
		StatusCode int
	}{
		{user1, password, title, http.StatusOK},
		{"admin", password, title, http.StatusOK},
		{user1, password, "project1", http.StatusBadRequest},
		{user1, password, "DoesNotExist", http.StatusBadRequest},
	} {
		// setup
		handler := NewProjectHandler()
		user, err := handler.DB.CreateAccount(user1, password, false)
		assert.NoError(err)
		_, err = handler.DB.CreateOrUpdateProject(user.Id, title)
		assert.NoError(err)

		r := NewTestRequest("POST", "/", nil, map[string]string{
			"title": s.Title,
		})
		r.SetBasicAuth(s.Username, s.Password)
		w := httptest.NewRecorder()

		handler.DeleteProject()(w, r)
		assert.Equal(s.StatusCode, w.Code)
	}
}
