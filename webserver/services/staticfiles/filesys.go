package staticfiles

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/otiai10/copy"
	"github.com/pkg/errors"

	"private-sphinx-docs/libs"
)

type FileSys struct {
	root string
}

func NewFileSys(root string) (*FileSys, error) {
	if !filepath.IsAbs(root) {
		_root, err := filepath.Abs(root)
		if err != nil {
			return nil, errors.Wrap(err, "could not convert error to absolute path")
		}
		root = _root
	}

	if !libs.PathExists(root) {
		err := os.MkdirAll(root, 0744)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create root folder at '%s'", root)
		}
	}

	return &FileSys{root}, nil
}

func (f *FileSys) Upload(r io.ReaderAt, name string, size int64) error {
	contents, err := zip.NewReader(r, size)
	if err != nil {
		return errors.Wrap(err, "could not read zip contents")
	}
	dest := f.Destination(name)
	err = os.RemoveAll(dest)
	if err != nil {
		return errors.Wrap(err, "could not remove old directory")
	}

	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(path, f.Mode())
		} else {
			_ = os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, file := range contents.File {
		if err := extractAndWriteFile(file); err != nil {
			return err
		}
	}

	return formatContentDirectory(dest)
}

func (f *FileSys) Destination(name string) string {
	name = strings.TrimSuffix(filepath.Base(name), ".zip")
	return filepath.Join(f.root, name)
}

func (f *FileSys) Remove(name string) error {
	return os.RemoveAll(f.Destination(name))
}

func (f *FileSys) Source() string {
	return f.root
}

// If the destination folder only contains 1 folder, moves the entire folder up 1
// level till we reach the first level with more than 1 item.
func formatContentDirectory(src string) error {
	root := src
	for {
		f, err := ioutil.ReadDir(src)
		if err != nil {
			return err
		}

		if len(f) != 1 {
			break
		}

		src = filepath.Join(src, filepath.Join(f[0].Name()))
	}

	if src != root {
		err := copy.Copy(src, root)
		if err != nil {
			return err
		}
		err = os.RemoveAll(src)
		if err != nil {
			return errors.Wrap(err, "could not remove old directory")
		}
	}

	return nil
}
