package server_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/chi"

	db "private-sphinx-docs/services/database"
)

func NewMockStore() *MockStore {
	project := &db.Project{
		Id:         1,
		Title:      "project1",
		LastUpdate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local),
		AccountId:  1,
	}

	account := &db.Account{
		Id:       1,
		Username: "admin",
		Password: "password",
		IsAdmin:  true,
		Projects: []*db.Project{project},
	}
	_ = account.SaltPassword()

	return &MockStore{
		accounts: map[string]*db.Account{account.Username: account},
		projects: map[string]*db.Project{project.Title: project},
	}
}

type MockStore struct {
	accounts map[string]*db.Account
	projects map[string]*db.Project
}

func (m *MockStore) FetchAccount(username string) (*db.Account, error) {
	acc, exist := m.accounts[username]
	if !exist {
		return nil, errors.New("account does not exist")
	}
	return acc, nil
}

func (m *MockStore) FetchAccounts() ([]*db.Account, error) {
	var accounts []*db.Account
	for _, a := range m.accounts {
		accounts = append(accounts, a)
	}

	return accounts, nil
}

func (m *MockStore) CreateAccount(username, password string, isAdmin bool) (*db.Account, error) {
	if _, exist := m.accounts[username]; exist {
		return nil, errors.New("account exists")
	}
	acc := &db.Account{
		Id:       len(m.accounts) + 1,
		Username: username,
		Password: password,
		IsAdmin:  isAdmin,
	}
	err := acc.Validate()
	if err != nil {
		return nil, err
	}

	err = acc.SaltPassword()
	if err != nil {
		return nil, err
	}
	m.accounts[username] = acc

	return acc, nil
}

func (m *MockStore) UpdateAccount(account *db.Account) (*db.Account, error) {
	err := account.Validate()
	if err != nil {
		return nil, err
	}
	acc, err := m.fetchAccount(account.Id)
	if err != nil {
		return nil, err
	}

	account.Id = acc.Id
	account.Projects = acc.Projects
	m.accounts[account.Username] = account
	return account, nil
}

func (m *MockStore) fetchAccount(id int) (*db.Account, error) {
	for _, acc := range m.accounts {
		if acc.Id == id {
			return acc, nil
		}
	}
	return nil, errors.New("account does not exist")
}

func (m *MockStore) DeleteAccount(username string) (*db.Account, error) {
	acc, exist := m.accounts[username]
	if !exist {
		return nil, errors.New("account does not exist")
	}

	delete(m.accounts, username)
	return acc, nil
}

func (m *MockStore) FetchProjects() ([]*db.Project, error) {
	var projects []*db.Project

	for _, p := range m.projects {
		projects = append(projects, p)
	}
	return projects, nil
}

func (m *MockStore) FetchProjectsByAccount(accountId int) ([]*db.Project, error) {
	acc, err := m.fetchAccount(accountId)
	if err != nil {
		return nil, err
	}

	return acc.Projects, nil
}

func (m *MockStore) fetchProject(title string) (*db.Project, error) {
	p, exist := m.projects[title]
	if !exist {
		return nil, errors.New("project does not exist")
	}
	return p, nil
}

func (m *MockStore) CreateOrUpdateProject(accountId int, title string) (*db.Project, error) {
	acc, err := m.fetchAccount(accountId)
	if err != nil {
		return nil, err
	}

	proj, err := m.fetchProject(title)
	if err != nil {
		// project does not exist, insert (create) it
		proj = &db.Project{
			Id:         len(m.projects) + 1,
			Title:      title,
			LastUpdate: time.Now(),
			AccountId:  accountId,
		}
		acc.Projects = append(acc.Projects, proj)
	} else {
		// Project does exist
		// Remove project from old account first if account ids are different
		if proj.AccountId != accountId {
			oldAcc, err := m.fetchAccount(proj.AccountId)
			if err != nil {
				return nil, errors.New("old account does not exist")
			}
			var projects []*db.Project
			for _, p := range oldAcc.Projects {
				if p.Id != proj.Id {
					projects = append(projects, p)
				}
			}
			oldAcc.Projects = projects
			m.accounts[oldAcc.Username] = oldAcc
		}

		// update changes in project object
		proj.AccountId = accountId

		// update the projects slice in the accounts object
		var projects []*db.Project
		for _, p := range acc.Projects {
			if p.Id == proj.Id {
				projects = append(projects, proj)
			} else {
				projects = append(projects, p)
			}
		}
		acc.Projects = projects

	}
	m.accounts[acc.Username] = acc
	m.projects[proj.Title] = proj

	return proj, nil
}

func (m *MockStore) DeleteProject(title string) error {
	if _, err := m.fetchProject(title); err != nil {
		return err
	}
	delete(m.projects, title)
	return nil
}

func (m *MockStore) CanOwnProject(accountId int, title string) (bool, error) {
	p, err := m.fetchProject(title)
	if err != nil {
		return true, nil
	}
	return p.AccountId == accountId, nil
}

func NewFileHandler() *MockFileHandler {
	return &MockFileHandler{}
}

type MockFileHandler struct {
}

func (m *MockFileHandler) Upload(r io.ReaderAt, name string, size int64) error {
	return nil
}

func (m *MockFileHandler) Destination(name string) string {
	return ""
}

func (m *MockFileHandler) Remove(name string) error {
	return nil
}

func (m *MockFileHandler) Source() string {
	return "source"
}

func NewTestRequest(method, target string, body io.Reader, routeParams map[string]string) *http.Request {
	r := httptest.NewRequest(method, target, body)

	if len(routeParams) > 0 {
		routeCtx := chi.NewRouteContext()
		for key, value := range routeParams {
			routeCtx.URLParams.Add(key, value)
		}

		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
	}

	return r
}
