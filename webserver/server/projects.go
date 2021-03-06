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
	Title string `json:"title"`
}

func (h *ProjectHandler) FetchProjects() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := strings.TrimSpace(chi.URLParam(r, "username"))

		if username == "" {
			projects, err := h.DB.FetchProjects()
			if err != nil {
				BadRequest(w, err)
				return
			}

			toJson(w, projects)
		} else {
			acc, err := h.DB.FetchAccount(username)
			if err != nil {
				BadRequest(w, err)
				return
			}

			projects, err := h.DB.FetchProjectsByAccount(acc.Id)
			if err != nil {
				BadRequest(w, err)
				return
			}
			toJson(w, projects)
		}
	}
}

func (h *ProjectHandler) UploadProject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account, err := authenticate(h.DB, r)
		if err != nil {
			Forbid(w, r)
			return
		}

		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		title := r.PostFormValue("title")
		err = h.canManageProject(account, title)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// save details in database
		project, err := h.DB.CreateOrUpdateProject(account.Id, title)
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

		err = h.FS.Upload(file, title, header.Size)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		toJson(w, project)
	}
}

func (h *ProjectHandler) DeleteProject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account, err := authenticate(h.DB, r)
		if err != nil {
			Forbid(w, r)
			return
		}

		title := chi.URLParam(r, "title")
		err = h.canManageProject(account, title)
		if err != nil {
			Forbid(w, r)
			return
		}

		err = h.DB.DeleteProject(title)
		if err != nil {
			BadRequest(w, err)
			return
		}

		err = h.FS.Remove(title)
		if err != nil {
			BadRequest(w, err)
			return
		}

		Ok(w, r)
	}
}

// check if the user can create, update or delete project
func (h *ProjectHandler) canManageProject(account *db.Account, title string) error {
	if account.IsAdmin {
		return nil
	}

	// check if the user can create or update project
	canOwn, err := h.DB.CanOwnProject(account.Id, title)
	if err != nil {
		return err
	} else if !canOwn {
		return errors.Errorf("user does not have rights to create/update project %s", title)
	}
	return nil
}
