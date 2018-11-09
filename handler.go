package summerfish

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileHandler returns an HTTP handler that serves the swagger.json file
func fileHandler(swaggerPath string) (http.Handler, error) {
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

func updateIndexFile(path string) (err error) {
	filePath, err := filepath.Abs("swaggerui/index.html")
	if err != nil {
		return
	}

	input, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(input), "\n")
	lines[76] = "url: \"" + path + "\","
	output := strings.Join(lines, "\n")
	return ioutil.WriteFile(filePath, []byte(output), 0644)
}
