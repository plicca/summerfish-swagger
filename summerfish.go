package summerfish

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

type Method map[string]Operation
type PathsHolder map[string]Method

type Config struct {
	Schemes                []string
	SwaggerFilePath        string
	SwaggerFileRoute       string
	SwaggerFileHeaderRoute string
	SwaggerUIRoute         string
	BaseRoute              string
}

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
	Items      *SchemaParameters           `json:"items,omitempty"`
	Properties map[string]SchemaParameters `json:"properties,omitempty"`
}

type RouteParserHolder struct {
	routeParsers []RouteParser
}

func GetInfoFromRouter(r *mux.Router) (holders []RouteHolder, err error) {
	routeParsers, err := getParsersFromRouter(r)
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

func getParsersFromRouter(r *mux.Router) (routeParsers []RouteParser, err error) {
	holder := RouteParserHolder{}
	err = r.Walk(holder.walkGorillaMuxRoutes)
	if err != nil {
		return
	}

	routeParsers = holder.routeParsers
	return
}

func (rph *RouteParserHolder) walkGorillaMuxRoutes(route *mux.Route, router *mux.Router, ancestors []*mux.Route) (err error) {
	pathTemplate, err := route.GetPathTemplate()
	if err != nil {
		if err.Error() == "mux: route doesn't have a path" {
			err = nil
		}

		return
	}

	methods, err := route.GetMethods()
	if err != nil {
		if err.Error() != "mux: route doesn't have methods" {
			return
		}

		err = nil
	}

	handler := route.Name(pathTemplate).GetHandler()
	if handler == nil {
		return
	}

	relativePath, fullPath, lineNumber := processHandler(handler)
	if lineNumber == 0 {
		return
	}

	rph.routeParsers = append(rph.routeParsers, RouteParser{
		Route:        pathTemplate,
		RelativePath: relativePath,
		FullPath:     fullPath,
		LineNumber:   lineNumber,
		Methods:      methods,
	})
	return
}

func generateFileMap(routeParsers []RouteParser) (sourceFiles map[string][]string, err error) {
	sourceFiles = make(map[string][]string)
	for _, rp := range routeParsers {
		_, wasProcessed := sourceFiles[rp.FullPath]
		if wasProcessed {
			continue
		}

		var lines []string
		lines, err = processRouteParserSourceFile(rp.FullPath)
		if err != nil {
			return
		}

		sourceFiles[rp.FullPath] = lines
	}
	return
}

func processRouteParserSourceFile(path string) (lines []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)
	isCommentSection := false
	var line string
	for scanner.Scan() {
		//clean commented lines or sections
		line, isCommentSection = cleanCommentSection(scanner.Text(), isCommentSection)
		lines = append(lines, line)
	}

	return
}

func cleanCommentSection(line string, commentSection bool) (string, bool) {
	if commentSection {
		if !strings.Contains(line, "*/") {
			return "", commentSection
		}
		line = strings.SplitN(line, "*/", 2)[1]
		commentSection = false
	}

	for {
		if !strings.Contains(line, "/*") {
			break
		}
		line, commentSection = RemoveCommentSection(line)
	}

	if strings.Contains(line, "//") {
		line = strings.SplitN(line, "//", 2)[0]
	}

	return line, commentSection
}

func RemoveCommentSection(line string) (string, bool) {
	lineSections := strings.SplitN(line, "/*", 2)
	line = lineSections[0]
	if strings.Contains(lineSections[1], "*/") {
		line = line + "" + strings.SplitN(lineSections[1], "*/", 2)[1]
		return line, false
	}

	return line, true
}

func (s *SchemeHolder) GenerateSwaggerFile(routes []RouteHolder, filePath string) (err error) {
	s.SwaggerVersion = "2.0"
	s.Paths = mapRoutesToPaths(routes)
	encoded, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}

	f, err := os.Create(filePath)
	if err != nil {
		return
	}

	defer f.Close()
	_, err = f.Write(encoded)
	return
}
