package summerfish

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type handler struct {
	modTime time.Time
	body    io.ReadSeeker
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeContent(w, r, "Swagger json spec", h.modTime, h.body)
}

// FileHandler returns an HTTP handler that serves the swagger.json file
func fileHandler(swaggerPath string) (http.Handler, error) {
	data, err := os.Open(swaggerPath)
	if err != nil {
		return nil, err
	}
	return &handler{modTime: time.Now(), body: data}, nil
}

func updateIndexFile(path, route string) (err error) {
	filePath := path + "/swaggerui/index.html"
	input, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(input), "\n")
	lines[41] = "url: \"" + route + "\","
	output := strings.Join(lines, "\n")
	return ioutil.WriteFile(filePath, []byte(output), 0644)
}
