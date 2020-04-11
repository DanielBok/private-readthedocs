package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	db "private-sphinx-docs/services/database"
)

type ProjectHandler struct {
	DB IStore
	FS IFileHandler
}

type DeleteProjectPayload struct {
	Requester *Credentials `json:"requester"`
	Title     string       `json:"title"`
}

func (h *ProjectHandler) FetchProjects() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := strings.TrimSpace(chi.URLParam(r, "username"))

		if username == "" {
			projects, err := h.DB.FetchProjects()
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}

			toJson(w, projects)
		} else {
			acc, err := h.DB.FetchAccount(username)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}

			projects, err := acc.FetchProjects()
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			toJson(w, projects)
		}
	}
}

func (h *ProjectHandler) UploadProject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		username := r.PostFormValue("username")
		password := r.PostFormValue("password")
		title := r.PostFormValue("title")

		acc, err := h.hasValidAccount(username, password)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		err = h.ownsProject(acc.Id, title)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// save details in database
		project, err := h.DB.CreateOrUpdateProject(acc.Id, title)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// upload static files
		file, header, err := r.FormFile("content")
		if err != nil {
			http.Error(w, errors.Wrap(err, "error retrieving file").Error(), 400)
			return
		}
		defer func() { _ = file.Close() }()

		err = h.FS.Upload(file, header.Filename, header.Size)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		toJson(w, project)
	}
}

func (h *ProjectHandler) DeleteProject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p *DeleteProjectPayload
		err := readJson(r, &p)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		acc, err := h.hasValidAccount(p.Requester.Username, p.Requester.Password)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		err = h.ownsProject(acc.Id, p.Title)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		err = h.DB.DeleteProject(p.Title)
		if err != nil {
			http.Error(w, errors.Wrap(err, "could not delete project").Error(), 400)
			return
		}

		err = h.FS.Remove(p.Title)
		if err != nil {
			http.Error(w, errors.Wrap(err, "could not delete static files").Error(), 400)
			return
		}

		ok(w)
	}
}

func (h *ProjectHandler) hasValidAccount(username, password string) (*db.Account, error) {
	acc, err := h.DB.FetchAccount(username)
	if err != nil {
		return nil, err
	}

	if !acc.HasValidPassword(password) {
		return nil, errors.New("invalid credentials")
	}

	return acc, nil
}

// check if the user can create, update or delete project
func (h *ProjectHandler) ownsProject(id int, title string) error {
	// check if the user can create or update project
	canOwn, err := h.DB.CanOwnProject(id, title)
	if err != nil {
		return err
	} else if !canOwn {
		return errors.Errorf("user does not have rights to create/update project %s", title)
	}
	return nil
}
