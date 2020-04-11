package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"

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
	fileServer := http.FileServer(http.Dir(h.Root))

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(ctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, fileServer)
		fs.ServeHTTP(w, r)
	}
}
