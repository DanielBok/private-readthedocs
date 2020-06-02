// This file is auto-generated and should not be edited by hand
package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"private-sphinx-docs/libs"
)

// Generates the migration files if they do not exist with the binary. Returns the source url
// used for migrate command if successful
func generateMigrationFiles() (string, error) {
	_, file, _, _ := runtime.Caller(0)
	wd := filepath.Dir(file)

	folder := filepath.Join(wd, "migrations")
	err := os.MkdirAll(folder, os.ModePerm)
	if err != nil {
		return "", err
	}

	_migrations := map[string]string{
		"01_accounts": `CREATE TABLE account
(
    id       SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE CHECK ( length(username) >= 4 ) NOT NULL,
    password VARCHAR(255) CHECK ( length(password) >= 4 )        NOT NULL,
    is_admin BOOLEAN DEFAULT FALSE
);

CREATE TABLE project
(
    id          SERIAL PRIMARY KEY,
    title       VARCHAR(255) UNIQUE CHECK ( length(title) >= 1 ) NOT NULL,
    last_update TIMESTAMP DEFAULT NOW(),
    account_id  INT REFERENCES account (id) ON UPDATE CASCADE ON DELETE CASCADE
);
`,
	}

	n := 0
	for title, content := range _migrations {
		fp := filepath.Join(folder, fmt.Sprintf("%s.up.sql", title))

		if !libs.PathExists(fp) {
			f, err := os.Create(fp)
			if err != nil {
				return "", errors.Wrapf(err, "error creating file: %s", title)
			}
			_, err = f.WriteString(content)
			if err != nil {
				return "", errors.Wrapf(err, "error writing content '%s' to file: %s", content, title)
			}
			n++
		}
	}

	log.Printf("generated %d/%d migration scripts", n, len(_migrations))
	log.Printf("note: it is okay not to generate migration scripts, it just means it already exists")
	log.Printf("applying %d levels of migration", len(_migrations))

	return fmt.Sprintf("file://%s", strings.Replace(folder, `\`, "/", -1)), nil
}
