package main

import (
	"log"
	"github.com/gorilla/mux"
	"path/filepath"
	"github.com/plicca/summerfish-swagger"
	"net/http"
	"fmt"
)

const port = ":8080"


func main() {
	router := mux.NewRouter()
	router.HandleFunc("/test/{tokenId}", GetStoryAuthorization).Methods("GET")
	err := GenerateSwaggerDocsAndEndpoints(router, "localhost" + port)
	if err != nil {
		fmt.Println(err)
		return
	}

	http.ListenAndServe(port, router)
}

func GetStoryAuthorization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tokenID := vars["tokenId"]
	w.Write([]byte(tokenID))
}

func GenerateSwaggerDocsAndEndpoints(router *mux.Router, endpoint string) (err error) {
	swaggerPath, err :=filepath.Abs("example/swagger.json")
	if err != nil {
		return
	}

	routerInformation, err := summerfish.GetInfoFromRouter(router)
	if err != nil {
		return
	}

	scheme := summerfish.SchemeHolder{Schemes: []string{"http", "https"}, Host: endpoint, BasePath: "/"}
	err = scheme.GenerateSwaggerFile(routerInformation, swaggerPath)
	if err != nil {
		return
	}

	log.Println("Swagger documentation generated")
	return summerfish.AddSwaggerUIEndpoints(router, swaggerPath)
}
