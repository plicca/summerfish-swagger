package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/plicca/summerfish-swagger"
)

const port = ":8080"

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/ping/", IsAlive).Methods("GET")
	router.HandleFunc("/upload/", UploadImage).Methods("POST")
	err := GenerateSwaggerDocsAndEndpoints(router, "localhost"+port)
	if err != nil {
		log.Println(err)
		return
	}

	http.ListenAndServe(port, router)
}

func UploadImage(w http.ResponseWriter, r *http.Request) {
	// Read image
	encoded, header, err := r.FormFile("image")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	additionalParams := r.FormValue("params")
	if len(additionalParams) > 0 {
		log.Println(additionalParams)
	}

	// Will contain filename and extension
	imageName := strings.Split(header.Filename, ".")
	log.Println(imageName)
	decoded, err := ioutil.ReadAll(encoded)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(decoded)
}

func IsAlive(w http.ResponseWriter, r *http.Request) {
	pingName := r.URL.Query().Get("pingID")
	log.Println("Got: ", pingName)
	w.Write([]byte("pong"))
}

func GenerateSwaggerDocsAndEndpoints(router *mux.Router, endpoint string) (err error) {
	routerInformation, err := summerfish.GetInfoFromRouter(router)
	if err != nil {
		return
	}

	scheme := summerfish.SchemeHolder{Schemes: []string{"http", "https"}, Host: endpoint, BasePath: "/", Information: summerfish.SchemeInformation{Title: "SummerFish Demo", Version: "0.0.1"}}

	swaggerFilePathYaml, err := filepath.Abs("swaggerui/swagger.yaml")
	if err != nil {
		return
	}

	err = scheme.GenerateSwaggerYaml(routerInformation, swaggerFilePathYaml)
	if err != nil {
		return
	}

	log.Println("Swagger documentation generated")
	swaggerUIRoute := "/docs/"
	router.PathPrefix(swaggerUIRoute).Handler(http.StripPrefix(swaggerUIRoute, http.FileServer(http.Dir("swaggerui/"))))
	return
}
