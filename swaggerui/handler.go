package swaggerui

import (
	"io"
	"net/http"
	"os"
	"github.com/plicca/summerfish-swagger/swaggerui/assetfs"
	"time"
)

const SwaggerPath = "/swagger.json"

// FileHandler returns an HTTP handler that serves the swagger.json file
func FileHandler(swaggerPath string) (http.Handler, error) {
	data, err := os.Open(swaggerPath)
	if err != nil {
		return nil, err
	}
	return &handler{modTime: time.Now(), body: data}, nil
}

// UIHandler returns an HTTP handler that serves the swagger UI
func UIHandler() (http.Handler, error) {
	assetStore := assetfs.CreateAssetStore()

	fs, err := assetfs.New(assetStore)
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FileSystem(fs)), nil
}

type handler struct {
	modTime time.Time
	body    io.ReadSeeker
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeContent(w, r, SwaggerPath, h.modTime, h.body)
}
