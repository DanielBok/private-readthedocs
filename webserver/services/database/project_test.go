package database_test

import (
	"testing"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	. "private-sphinx-docs/services/database"
)

const project1 = "Project1"

func TestNewProject(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	for _, r := range []struct {
		Title     string
		AccountId int
		HasError  bool
	}{
		{"package", 1, false},
		{"package", 0, true},
		{"p", 1, true},
	} {
		proj, err := NewProject(r.Title, r.AccountId)
		if r.HasError {
			assert.Error(err)
		} else {
			assert.NoError(err)
			assert.IsType(&Project{}, proj)
		}
	}
}

func TestDatabase_CreateProject(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	projects, err := mockProjects()
	assert.NoError(err)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)
		defer closeDb(db)

		acc, err := db.FetchAccount(admin)
		assert.NoError(err)

		for _, p := range projects {
			p.AccountId = acc.Id
			proj, err := db.CreateProject(p)
			assert.NoError(err)
			assert.IsType(&Project{}, proj)
		}

		_, err = db.CreateProject(projects[0])
		assert.Error(err, "project already exists")

		// test that validation raises errors
		_, err = db.CreateProject(&Project{
			Title:     "",
			AccountId: acc.Id,
		})
		assert.Error(err, "validation failed")
	})
}

func TestDatabase_FetchProject(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedProjects)
		assert.NoError(err)
		defer closeDb(db)

		for _, r := range []struct {
			Title    string
			HasError bool
		}{
			{project1, false},
			{"DoesNotExist", true},
		} {
			proj, err := db.FetchProject(r.Title)
			if r.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.IsType(&Project{}, proj)
			}
		}
	})
}

func TestDatabase_FetchUserProjects(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedProjects)
		assert.NoError(err)
		defer closeDb(db)

		acc, err := db.FetchAccount(admin)
		assert.NoError(err)

		projects, err := db.FetchProjectsByAccount(acc.Id)
		assert.NoError(err)
		assert.Greater(len(projects), 0)
	})
}

func TestDatabase_FetchProjects(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedProjects)
		assert.NoError(err)
		defer closeDb(db)

		projects, err := db.FetchProjects()
		assert.NoError(err)

		mocks, _ := mockProjects()
		assert.Len(projects, len(mocks))
	})
}

func TestDatabase_UpdateProject(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedProjects)
		assert.NoError(err)
		defer closeDb(db)

		proj, err := db.FetchProject(project1)
		assert.NoError(err)

		_, err = db.UpdateProject(&Project{
			Id:        proj.Id,
			Title:     "NewName",
			AccountId: proj.AccountId,
		})
		assert.NoError(err)
	})
}

func TestDatabase_CreateOrUpdateProject(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedProjects)
		assert.NoError(err)
		defer closeDb(db)

		acc, err := db.FetchAccount(admin)
		assert.NoError(err)

		for _, r := range []struct {
			AccountId int
			Title     string
			HasError  bool
		}{
			{acc.Id, "NewProject", false}, // create
			{acc.Id, project1, false},     // update
			{acc.Id + 1, project1, true},  // cannot update
			{acc.Id + 99, project1, true}, // cannot create
		} {
			_, err := db.CreateOrUpdateProject(r.AccountId, r.Title)
			if r.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		}
	})
}

func TestDatabase_DeleteProject(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedProjects)
		assert.NoError(err)
		defer closeDb(db)

		for _, r := range []struct {
			Name     string
			HasError bool
		}{
			{project1, false},
			{"DoesNotExist", true},
		} {
			err := db.DeleteProject(r.Name)
			if r.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		}
	})
}

func TestDatabase_CanOwnProject(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts, seedProjects)
		assert.NoError(err)
		defer closeDb(db)

		for _, r := range []struct {
			Username string
			Title    string
			Expected bool
		}{
			{admin, project1, true},
			{user1, project1, false},
			{user1, "NewProject", true},
		} {
			acc, err := db.FetchAccount(r.Username)
			assert.NoError(err)

			actual, err := db.CanOwnProject(acc.Id, r.Title)
			assert.NoError(err)
			assert.Equal(r.Expected, actual)
		}
	})
}

// Utilities here
func mockProjects() ([]*Project, error) {
	var projects []*Project

	for _, v := range []struct {
		Title     string
		AccountId int
	}{
		{project1, 1},
		{"Project2", 1},
	} {
		d, err := NewProject(v.Title, v.AccountId)
		if err != nil {
			return nil, err
		}
		projects = append(projects, d)
	}
	return projects, nil
}

func seedProjects(db *Database) error {
	projects, err := mockProjects()
	if err != nil {
		return err
	}

	acc, err := db.FetchAccount(admin)
	if err != nil {
		return err
	}

	for _, d := range projects {
		d.AccountId = acc.Id
		_, err = db.CreateProject(d)
		if err != nil {
			return err
		}
	}
	return nil
}
