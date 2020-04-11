package staticfiles

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	return nil
}

func (f *FileSys) Destination(name string) string {
	name = strings.TrimSuffix(filepath.Base(name), ".zip")
	return filepath.Join(f.root, name)
}

func (f *FileSys) Remove(name string) error {
	return os.RemoveAll(f.Destination(name))
}
