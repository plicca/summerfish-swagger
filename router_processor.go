package summerfish

import (
	"bufio"
	"go/build"
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
	Name     string
	Type     string
	Children []NameType
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
				rh.Path[i] = rp.searchForAll(rh.Path[i].Name, lines)
			}

			for i := range rh.Query {
				rh.Query[i] = rp.searchForAll(rh.Query[i].Name, lines)
			}
			if len(rh.Body.Name) > 0 {
				rh.Body = rp.searchForAll(rh.Body.Name, lines)
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

func (rp *RouteParser) searchForAll(name string, lines []string) NameType {
	varType := rp.searchForType(name, lines)
	if len(varType) == 0 {
		return NameType{name, "string", nil}
	}

	if len(strings.Split(varType, ".")) <= 1 {
		return NameType{name, varType, nil}
	}

	candidateSourceFiles, err := rp.searchForFullPath(varType, lines)
	if err != nil {
		return NameType{name, "", nil}
	}

	if len(candidateSourceFiles) > 0 {

		//search struct, all the children and children of children, and chil....

		//fullStruct := rp.searchForStruct(varType, candidateSourceFiles)
		//return strings.Join(fullStruct, ",")
		return rp.searchForStruct(varType, candidateSourceFiles)
	}

	return NameType{name, "", nil}
}

func (rp *RouteParser) searchForStruct(name string, paths []string) (result NameType) {
	structName := strings.Split(name, ".")[1]
	comp := "type " + structName + " struct"
	exp := "\\s*\\w+\\s+\\b(.+)\\b\\s+\\S*\\s*json:\"(.+)\""
	bodyTypeRegex, _ := regexp.Compile(exp)

	result.Name = structName

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
					var varName string
					var varType string

					splitResult := strings.Split(typeResult[2], ",")
					if len(splitResult) > 1 {
						varName = splitResult[0]
					}

					varType = typeResult[1]
					//se o tipo da variavel tiver um ponto é porque é uma struct é preciso ir procurar os seus filhos
					if strings.Contains(varType, ".") {
						//search for structure childrens
					} else {
						result.Children = append(result.Children, NameType{varName, varType, nil})
					}
				}
			} else if strings.HasPrefix(lineText, comp) {
				isFound = true
			}
		}
		file.Close()
	}

	//this means it doesn't found the struct in the candidated files. We should search the packages again!
	return
}

func (rp *RouteParser) searchForType(name string, lines []string) string {
	exp := "var " + name + " (.+)"
	exp2 := name + " := (.+){"

	bodyTypeRegex, _ := regexp.Compile(exp)
	bodyTypeRegex2, _ := regexp.Compile(exp2)
	for i := rp.LineNumber; i < len(lines); i++ {
		lineText := lines[i]
		typeResult := bodyTypeRegex.FindStringSubmatch(lineText)
		if len(typeResult) > 1 {
			return typeResult[1]
		}

		typeResult = bodyTypeRegex2.FindStringSubmatch(lineText)
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
			goPath := os.Getenv("GOPATH")
			if len(goPath) == 0 {
				goPath = build.Default.GOPATH
			}

			var fullPath string
			fullPath, err = filepath.Abs(goPath + "/src/" + path[1] + "/" + splitName)
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
					result = append(result, fullPath+"/"+file.Name())
				}
			}
			return
		}
	}
	return
}
