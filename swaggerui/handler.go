package swaggerui

import (
	"io"
	"net/http"
	"os"
	"time"
)

// FileHandler returns an HTTP handler that serves the swagger.json file
func FileHandler(swaggerPath string) (http.Handler, error) {
	data, err := os.Open(swaggerPath)
	if err != nil {
		return nil, err
	}
	return &handler{modTime: time.Now(), body: data}, nil
}

type handler struct {
	modTime time.Time
	body    io.ReadSeeker
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeContent(w, r, "Swagger json spec", h.modTime, h.body)
}
