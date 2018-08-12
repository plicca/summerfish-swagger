package swagger

import (
	"bufio"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

type RouteParser struct {
	Route        string
	RelativePath string
	FullPath     string
	LineNumber   int
	Methods      []string
}

type RouteHolder struct {
	Path    []NameType
	Query   []NameType
	Body    NameType
	Route   string
	Methods []string
	Name    string
}

type NameType struct {
	Name string
	Type string
}

func (rp *RouteParser) processHandler(handler http.Handler) {
	ptr := runtime.FuncForPC(reflect.ValueOf(handler).Pointer())
	rp.RelativePath = ptr.Name()
	rp.FullPath, rp.LineNumber = runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).FileLine(ptr.Entry())
	return
}

func (rp *RouteParser) processSourceFiles(lines []string) (rh RouteHolder) {

	pathRegex, _ := regexp.Compile("vars\\[\"(.+?)\"\\]")
	queryRegex, _ := regexp.Compile("r\\.URL\\.Query\\(\\).Get\\(\"(.+)\"\\)")
	bodyRegex, _ := regexp.Compile("json.NewDecoder\\(r.Body\\).Decode\\((.+)\\)")

	rh.Route = rp.Route
	rh.Methods = rp.Methods
	rh.Name = strings.Split(rp.RelativePath, ".")[1]

	for i := rp.LineNumber; i < len(lines); i++ {
		lineText := lines[i]

		//Finds the end of the function
		if lineText == "}" {
			for i := range rh.Path {
				rh.Path[i].Type = rp.searchForAll(rh.Path[i].Name, lines)
			}

			for i := range rh.Query {
				rh.Query[i].Type = rp.searchForAll(rh.Query[i].Name, lines)
			}
			if len(rh.Body.Name) > 0 {
				rh.Body.Type = rp.searchForAll(rh.Body.Name, lines)
			}
			return
		}

		pathResult := pathRegex.FindStringSubmatch(lineText)
		if len(pathResult) > 1 {
			rh.Path = append(rh.Path, NameType{Name: pathResult[1]})
		}

		queryResult := queryRegex.FindStringSubmatch(lineText)
		if len(queryResult) > 1 {
			name := strings.Replace(queryResult[1], "\"", "", -1)
			rh.Query = append(rh.Query, NameType{Name: name})
		}

		bodyResult := bodyRegex.FindStringSubmatch(lineText)
		if len(bodyResult) > 1 {
			rh.Body.Name = strings.Replace(bodyResult[1], "&", "", 1)
		}
	}
	return
}

func (rp *RouteParser) searchForAll(name string, lines []string) string {
	varType := rp.searchForType(name, lines)
	if len(varType) == 0 {
		return "string"
	}

	if len(strings.Split(varType, ".")) <= 1 {
		return varType
	}

	candidateSourceFiles, err := rp.searchForFullPath(varType, lines)
	if err != nil {
		return ""
	}

	if len(candidateSourceFiles) > 0 {
		fullStruct := rp.searchForStruct(varType, candidateSourceFiles)
		return strings.Join(fullStruct, ",")
	}

	return ""
}

func (rp *RouteParser) searchForStruct(name string, paths []string) (result []string) {
	comp := "type " + strings.Split(name, ".")[1] + " struct"
	exp := "json:\"(.+)\""
	bodyTypeRegex, _ := regexp.Compile(exp)
	for _, path := range paths {
		isFound := false
		var file *os.File
		file, _ = os.Open(path)

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lineText := scanner.Text()
			if isFound {
				if lineText == "}" {
					file.Close()
					return
				}
				typeResult := bodyTypeRegex.FindStringSubmatch(lineText)
				if len(typeResult) > 1 {
					splitResult := strings.Split(typeResult[1], ",")
					if len(splitResult) > 1 {
						typeResult[1] = splitResult[0]
					}
					result = append(result, typeResult[1])
				}
			} else if strings.HasPrefix(lineText, comp) {
				isFound = true
			}
		}
		file.Close()
	}
	return
}

func (rp *RouteParser) searchForType(name string, lines []string) string {
	exp := "var " + name + " (.+)"
	bodyTypeRegex, _ := regexp.Compile(exp)
	for i := rp.LineNumber; i < len(lines); i++ {
		lineText := lines[i]
		typeResult := bodyTypeRegex.FindStringSubmatch(lineText)
		if len(typeResult) > 1 {
			return typeResult[1]
		}
	}
	return ""
}

func (rp *RouteParser) searchForFullPath(name string, lines []string) (result []string, err error) {
	splitName := strings.Split(name, ".")[0]
	exp := "\"(.+)/" + splitName
	regex, _ := regexp.Compile(exp)
	for i := 0; i < len(lines); i++ {
		lineText := lines[i]
		if strings.HasPrefix(lineText, "func") {
			return
		}

		path := regex.FindStringSubmatch(lineText)
		if len(path) > 1 {
			//TODO review how to get true path
			var fullPath string
			fullPath, err = filepath.Abs("../../" + path[1] + "/" + splitName)
			if err != nil {
				return
			}

			var files []os.FileInfo
			files, err = ioutil.ReadDir(fullPath)
			if err != nil {
				return
			}

			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".go") {
					path = append(path, fullPath+"/"+file.Name())
				}
			}
			return
		}
	}
	return
}
