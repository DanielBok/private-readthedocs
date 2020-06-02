package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"private-sphinx-docs/libs"
)

type DocumentationHandler struct {
	Root string
}

func (h *DocumentationHandler) MustInit() {
	if !libs.PathExists(h.Root) {
		err := os.MkdirAll(h.Root, 0744)
		if err != nil {
			panic(err)
		}
	}
}

func (h *DocumentationHandler) FileServer() http.HandlerFunc {
	log.Infof("Serving doc files from root folder: %s", h.Root)

	return func(w http.ResponseWriter, r *http.Request) {
		// The package name
		name := strings.Split(r.Host, ".")[0]
		root := http.Dir(filepath.Join(h.Root, name))
		fileServer := http.FileServer(root)

		ctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(ctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, fileServer)
		fs.ServeHTTP(w, r)
	}
}
