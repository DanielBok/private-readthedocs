from pathlib import Path
from subprocess import run, DEVNULL

folder = Path(__file__).parent.parent.joinpath('services', 'database')

template = """
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
	    /*CONTENT*/
	}

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
		}
	}

	log.Printf("generated %d migration scripts", len(_migrations))
	
	return fmt.Sprintf("file://%s", strings.Replace(folder, `\`, "/", -1)), nil
}
""".strip()


def read_migration_content():
    contents = []
    for script in folder.joinpath('migrations').glob('*.up.sql'):
        name = script.name.split('.')[0]
        with open(script.absolute().as_posix()) as f:
            contents.append(f'"{name}": `{f.read()}`,')

    return '\n'.join(contents)


def write_migration_file():
    content = template
    for key, value in {
        "/*CONTENT*/": read_migration_content(),
    }.items():
        content = content.replace(key, value)

    file = Path(__file__).parent.parent.joinpath('services', 'database', 'migrations.go')
    with open(file, 'w') as f:
        f.write(content)

    run(['go', 'fmt', file.absolute().as_posix()], stdout=DEVNULL)
    print("Generated migration files")


if __name__ == '__main__':
    write_migration_file()
