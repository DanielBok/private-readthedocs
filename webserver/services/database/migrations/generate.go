package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"private-sphinx-docs/libs"
)

const template = `// This file is auto-generated and should not be edited by hand
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

	_migrations := map[string]string{%s}

	for title, content := range _migrations {
		fp := filepath.Join(folder, fmt.Sprintf("%%s.up.sql", title))

		if !libs.PathExists(fp) {
			f, err := os.Create(fp)
			if err != nil {
				return "", errors.Wrapf(err, "error creating file: %%s", title)
			}
			_, err = f.WriteString(content)
			if err != nil {
				return "", errors.Wrapf(err, "error writing content '%%s' to file: %%s", content, title)
			}
		}
	}

	log.Printf("generated %%d migration scripts", len(_migrations))

	return fmt.Sprintf("file://%%s", strings.Replace(folder, "\\", "/", -1)), nil
}
`

func main() {
	_, __file__, _, _ := runtime.Caller(0)
	wd := filepath.Dir(__file__)
	log.Printf("Running migration scripts generation from '%s'", wd)

	// prepare content
	fileInfos, err := ioutil.ReadDir(wd)
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not read files in migrations directory"))
	}

	components := make(map[string]string)
	for _, info := range fileInfos {
		if strings.HasSuffix(info.Name(), ".up.sql") {
			title := strings.Split(info.Name(), ".")[0]
			components[title] = readFile(filepath.Join(wd, info.Name()))
		}
	}

	// open file
	var file *os.File
	fp := filepath.Join(filepath.Dir(wd), "migrations.go")
	if libs.PathExists(fp) {
		file, err = os.OpenFile(fp, os.O_RDWR|os.O_TRUNC, 0644)
	} else {
		file, err = os.Create(fp)
	}
	if err != nil {
		log.Fatal("could not create/open migrations.go")
	}
	defer func() { _ = file.Close() }()

	// write template to file
	text := ""
	for title, content := range components {
		text += fmt.Sprintf(`"%s": `+"`%s`,", title, content)
	}
	text = strings.TrimSpace(text + "\n")

	_, err = file.WriteString(fmt.Sprintf(template, text))
	if err != nil {
		log.Fatal("could not write to migrations.go")
	}
	log.Print("Generated migrations successfully")
}

func readFile(path string) string {
	out, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("could not read file: %s", path)
	}
	return string(out)
}
