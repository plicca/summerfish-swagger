package summerfish

import (
	"bufio"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/plicca/summerfish-swagger/swaggerui"
	"os"
)

type Method map[string]Operation
type PathsHolder map[string]Method

type InputParameter struct {
	Type        string           `json:"type"`
	GoName      string           `json:"x-go-name"`
	Description string           `json:"description"`
	Name        string           `json:"name"`
	QueryType   string           `json:"in"`
	Schema      SchemaParameters `json:"schema"`
}

type Operation struct {
	Parameters []InputParameter  `json:"parameters"`
	ID         string            `json:"operationId"`
	Summary    string            `json:"summary"`
	Tags       []string          `json:"tags"`
	Responses  map[string]string `json:"responses"`
}

type SchemaParameters struct {
	Type       string                      `json:"type"`
	Properties map[string]SchemaParameters `json:"properties,omitempty"`
}

func GetInfoFromRouter(r *mux.Router) (holders []RouteHolder, err error) {
	var routeParsers []RouteParser
	err = r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) (err error) {
		routeParser := RouteParser{}
		routeParser.Route, err = route.GetPathTemplate()
		if err != nil {
			return
		}

		methods, err := route.GetMethods()
		if err != nil && err.Error() != "mux: route doesn't have methods" {
			return
		}

		handler := route.Name(routeParser.Route).GetHandler()
		if handler == nil {
			return
		}

		routeParser.Methods = methods
		routeParser.processHandler(handler)
		routeParsers = append(routeParsers, routeParser)
		return
	})
	if err != nil {
		return
	}

	sourceFiles, err := generateFileMap(routeParsers)
	if err != nil {
		return
	}

	for _, rp := range routeParsers {
		routeHolder := rp.processSourceFiles(sourceFiles[rp.FullPath])
		holders = append(holders, routeHolder)
	}

	return
}

func generateFileMap(routeParsers []RouteParser) (sourceFiles map[string][]string, err error) {
	sourceFiles = make(map[string][]string)
	for _, rp := range routeParsers {
		if _, wasProcessed := sourceFiles[rp.FullPath]; !wasProcessed {
			var file *os.File
			file, err = os.Open(rp.FullPath)
			if err != nil {
				return
			}

			var lines []string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}

			file.Close()
			sourceFiles[rp.FullPath] = lines
		}
	}
	return
}

func (s *SchemeHolder) GenerateSwaggerFile(routes []RouteHolder, filePath string) (err error) {
	s.Paths = mapRoutesToPaths(routes)
	encoded, err := json.Marshal(s)
	if err != nil {
		return
	}

	f, err := os.Create(filePath)
	if err != nil {
		return
	}

	defer f.Close()
	f.Write(encoded)
	return
}

func AddSwaggerUIEndpoints(router *mux.Router, swaggerPath string) (err error) {
	fileHandler, err := swaggerui.FileHandler(swaggerPath)
	if err != nil {
		return
	}

	uiHandler, err := swaggerui.UIHandler()
	if err != nil {
		return
	}

	router.Handle(swaggerui.SwaggerPath, fileHandler)
	router.PathPrefix("/swagger-ui/").Handler(uiHandler)
	return
}
