package database

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"
)

type Project struct {
	Id         int       `json:"id"`
	Title      string    `json:"title"`
	LastUpdate time.Time `json:"lastUpdate" db:"last_update"`
	AccountId  int       `json:"-" db:"account_id"`
}

func (p *Project) Validate() error {
	if len(p.Title) < 2 {
		return errors.New("project title must have 2 or more characters")
	} else if p.AccountId <= 0 {
		return errors.New("project must have valid account Id")
	}
	return nil
}

func NewProject(title string, accountId int) (*Project, error) {
	proj := &Project{
		Title:      title,
		LastUpdate: time.Now(),
		AccountId:  accountId,
	}

	if err := proj.Validate(); err != nil {
		return nil, err
	}

	return proj, nil
}

func (d *Database) FetchProject(title string) (*Project, error) {
	var err error
	tx := d.MustBegin()
	defer tx.Close(err)

	proj := &Project{}
	err = tx.Get(proj, `SELECT * FROM project WHERE title = $1`, title)
	if err != nil {
		return nil, err
	}

	return proj, nil
}

func (d *Database) FetchProjects(accountId int) ([]*Project, error) {
	var err error
	tx := d.MustBegin()
	defer tx.Close(err)

	query := "SELECT id, title, last_update, account_id FROM project"
	var projects []*Project
	if accountId <= 0 {
		return nil, errors.Errorf("Invalid account id '%d'. Use 0 if you want to query everything", accountId)
	} else if accountId == 0 {
		err = tx.Select(&projects, query)
		if err != nil {
			return nil, err
		}
		return projects, nil
	} else {
		err = tx.Select(&projects, query+" WHERE account_id = $1", accountId)
		if err != nil {
			return nil, err
		}
		return projects, nil
	}
}

func (d *Database) CreateOrUpdateProject(accountId int, title string) (*Project, error) {
	var err error

	tx := d.MustBegin()
	defer tx.Close(err)

	proj := &Project{}
	err = tx.Get(proj, `SELECT * FROM project WHERE title = $1 AND account_id = $2`, title, accountId)

	switch err {
	case sql.ErrNoRows:
		// No such project thus create it. If a project with a similar title but different account exists,
		// the insertion will run into an error again since title must be unique. Thus the user must check
		// that the project title does not belong to anyone else with d.CanOwnProject()
		return d.CreateProject(&Project{
			Title:     title,
			AccountId: accountId,
		})
	case nil:
		// Update project
		return d.UpdateProject(proj)
	default:
		return nil, err
	}
}

func (d *Database) CreateProject(project *Project) (*Project, error) {
	err := project.Validate()
	if err != nil {
		return nil, err
	}

	tx := d.MustBegin()
	defer tx.Close(err)

	project.LastUpdate = time.Now()
	rows, err := tx.NamedQuery(`
INSERT INTO project (title, last_update, account_id) 
VALUES (:title, :last_update, :account_id)
RETURNING id
`, project)
	if err != nil {
		return nil, err
	}

	project.Id = mustGetId(rows)

	return project, nil
}

func (d *Database) UpdateProject(project *Project) (*Project, error) {
	err := project.Validate()
	if err != nil {
		return nil, err
	}

	tx := d.MustBegin()
	defer tx.Close(err)

	project.LastUpdate = time.Now()
	n, err := tx.NamedExec(`
UPDATE project
SET title = :title,
    last_update = :last_update,
	account_id = :account_id
WHERE id = :id
`, project)
	if err != nil {
		return nil, err
	} else if n == 0 {
		return nil, errors.Errorf("no project with id: %d", project.Id)
	}

	return project, nil
}

func (d *Database) DeleteProject(title string) error {
	proj, err := d.FetchProject(title)
	if err != nil {
		return err
	}

	tx := d.MustBegin()
	defer tx.Close(err)

	_, err = tx.Exec(`DELETE FROM project WHERE id = $1`, proj.Id)

	return nil
}

// Verifies if the user can own this project. Project can only be owned if
// the project does not already belong to another user
func (d *Database) CanOwnProject(accountId int, title string) (bool, error) {
	proj, err := d.FetchProject(title)
	if err == sql.ErrNoRows {
		return true, nil
	} else if err != nil {
		return false, err
	}

	return proj.AccountId == accountId, nil
}
