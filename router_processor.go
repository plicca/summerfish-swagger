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

	kitHttp "github.com/go-kit/kit/transport/http"
)

type RouteParser struct {
	ID                   int
	Route                string
	RelativePath         string
	FullPath             string
	LineNumber           int
	Methods              []string
	IsOnlyEndpointParser bool
}

type RoutePath struct {
	RelativePath string
	FullPath     string
	LineNumber   int
}

type RouteHolder struct {
	ID       int
	Path     []NameType
	Query    []NameType
	Body     NameType
	FormData []NameType
	Route    string
	Methods  []string
	Name     string
}

type NameType struct {
	Name       string
	Type       string
	IsArray    bool
	Children   []NameType
	IsRequired bool
}

var nativeTypes = map[string]bool{
	"bool":       true,
	"string":     true,
	"int":        true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"uint":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"uintptr":    true,
	"byte":       true,
	"rune":       true,
	"float32":    true,
	"float64":    true,
	"complex64":  true,
	"complex128": true,
}

func processHandler(handler http.Handler) (RoutePath, RoutePath) {
	v, ok := handler.(*kitHttp.Server)
	if ok {
		ptrDecoder := reflect.ValueOf(v).Elem().FieldByName("dec").Pointer()
		ptrEndpoint := reflect.ValueOf(v).Elem().FieldByName("e").Pointer()
		return getRoutePathForPointer(ptrDecoder), getRoutePathForPointer(ptrEndpoint)
	} else {
		ptrHolder := reflect.ValueOf(handler).Pointer()
		return getRoutePathForPointer(ptrHolder), RoutePath{}
	}

	return RoutePath{}, RoutePath{}
}

func getRoutePathForPointer(ptrHolder uintptr) (rp RoutePath) {
	ptr := runtime.FuncForPC(ptrHolder)
	if ptr == nil {
		return
	}

	rp.RelativePath = ptr.Name()
	rp.FullPath, rp.LineNumber = ptr.FileLine(ptr.Entry())
	return
}

func (rp *RouteParser) processSourceFilesForEndpoint(lines []string) (rh RouteHolder) {
	returnRegex, _ := regexp.Compile(`return.*(\.|\s)\s?(.*)\(`)
	rh.Route = rp.Route
	rh.Methods = rp.Methods
	rh.ID = rp.ID

	for i := rp.LineNumber; i < len(lines); i++ {
		lineText := lines[i]

		//Finds the end of the function
		if lineText == "}" {
			return
		}

		trimedLine := strings.TrimSpace(lineText)
		if !strings.HasPrefix(trimedLine, "return") {
			continue
		}

		group := returnRegex.FindAllStringSubmatch(trimedLine, -1)
		if len(group) == 0 || len(group[0]) == 0 {
			continue
		}

		rh.Name = group[0][len(group[0])-1]
	}

	return
}

func (rp *RouteParser) processSourceFiles(lines []string) (rh RouteHolder) {
	functionNameRegex, _ := regexp.Compile(`func\s(\(.*\))?\s?(?U)(.*)\s?\(.*{`)
	pathRegex, _ := regexp.Compile(`vars\["(.+?)"\]`)
	queryRegex, _ := regexp.Compile(`r\.URL\.Query\(\).Get\("(.+)"\)`)
	bodyRegex, _ := regexp.Compile(`json.NewDecoder\(r.Body\).Decode\((.+)\)`)
	bodyFormFileRegex, _ := regexp.Compile(`r\.FormFile\("(.+)"\)`)
	bodyFormValueRegex, _ := regexp.Compile(`r\.FormValue\("(.+)"\)`)

	rh.Route = rp.Route
	rh.Methods = rp.Methods
	rh.ID = rp.ID

	if rp.LineNumber > 0 {
		functionNameResult := functionNameRegex.FindStringSubmatch(lines[rp.LineNumber-1])
		if len(functionNameResult) > 1 {
			rh.Name = functionNameResult[len(functionNameResult)-1]
		}
	}

	if len(rh.Name) == 0 {
		if strings.Contains(rp.RelativePath, "go-kit") {
			split := strings.Split(rp.Route, "/")
			rh.Name = split[len(split)-1]
		} else {
			rh.Name = strings.Split(rp.RelativePath, ".")[1]
		}
	}

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

		bodyFormResult := bodyFormFileRegex.FindStringSubmatch(lineText)
		if len(bodyFormResult) > 1 {
			rh.FormData = append(rh.FormData, NameType{Name: bodyFormResult[1], Type: "file"})
		}

		bodyFormValueResult := bodyFormValueRegex.FindStringSubmatch(lineText)
		if len(bodyFormValueResult) > 1 {
			rh.FormData = append(rh.FormData, NameType{Name: bodyFormValueResult[1], Type: "string"})
		}
	}
	return
}

func (rp *RouteParser) searchForAll(name string, lines []string) NameType {
	varType := rp.searchForType(name, lines)
	if len(varType) == 0 {
		return NameType{Name: name, Type: "string"}
	}

	_, ok := nativeTypes[varType]
	if ok {
		return NameType{Name: name, Type: jsonMapping[varType]}
	}

	var candidateSourceFiles = []string{}
	var err error
	if len(strings.Split(varType, ".")) <= 1 {
		varType, candidateSourceFiles, err = rp.searchCurrentPackage(varType)
	} else {
		candidateSourceFiles = rp.searchForFullPath(varType, lines)
	}
	if err != nil || len(candidateSourceFiles) == 0 {
		return NameType{Name: name, Type: ""}
	}

	return rp.searchForStruct(varType, "", candidateSourceFiles, false)
}

func (rp *RouteParser) searchForStruct(name string, childrenNameFromParent string, paths []string, isArray bool) (result NameType) {
	structInfo := strings.Split(name, ".")
	structPackage := structInfo[0]
	structName := structInfo[1]

	if len(childrenNameFromParent) > 0 {
		result.Name = childrenNameFromParent
	} else {
		result.Name = structName
	}

	result.IsArray = isArray
	for _, path := range paths {
		children, isFinished := rp.searchForStructInOneFile(path, structPackage, structName, paths)
		if len(children) > 0 {
			result.Children = append(result.Children, children...)
		}

		if isFinished {
			return
		}
	}

	return
}

func (rp *RouteParser) searchForStructInOneFile(path, structPackage, structName string, paths []string) (children []NameType, isFinished bool) {
	bodyTypeRegex, _ := regexp.Compile("^\\s*(.+)\\b\\s+(.+)\\b(\\s+`(.+)`)?$")
	formattedStructName := "type " + structName + " struct"

	file, err := os.Open(path)
	if err != nil {
		return
	}

	defer file.Close()
	commentSection := false
	isFound := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineText := scanner.Text()
		lineText, commentSection = cleanCommentSection(lineText, commentSection)
		if isFound {
			if lineText == "}" {
				isFinished = true
				return
			}

			typeResult := bodyTypeRegex.FindStringSubmatch(lineText)
			if len(typeResult) > 1 {
				children = append(children, rp.findNativeType(structPackage, typeResult[1], typeResult[2], typeResult[3], paths))
			}
		} else if strings.HasPrefix(lineText, formattedStructName) {
			isFound = true
		}
	}

	return
}

func (rp *RouteParser) findNativeType(structPackage string, varName, varType, varTags string, paths []string) (output NameType) {
	jsonTagRegex, _ := regexp.Compile(`(?U)json:"(.+)"`)
	if len(varTags) > 0 {
		jsonResults := jsonTagRegex.FindStringSubmatch(varTags)
		if len(jsonResults) > 1 {
			splitResult := strings.Split(jsonResults[1], ",")
			varName = splitResult[0]
		}
	}

	isArray := false

	//Array verification
	if strings.HasPrefix(varType, "[]") {
		isArray = true
		varType = strings.SplitN(varType, "]", 2)[1]
	}

	_, ok := nativeTypes[varType]
	if ok {
		return NameType{varName, varType, isArray, nil, false}
	}

	//appends package name if internal
	if !strings.Contains(varType, ".") {
		varType = strings.Join([]string{structPackage, varType}, ".")
	}

	return rp.searchForStruct(varType, varName, paths, isArray)
}

func (rp *RouteParser) searchForType(name string, lines []string) string {
	exp := "var " + name + " (.+)"
	exp2 := name + " := (.+){"
	exp3 := convertToCamelCase(name) + ".* := strconv\\.Parse(.*)\\("

	bodyTypeRegex, _ := regexp.Compile(exp)
	bodyTypeRegex2, _ := regexp.Compile(exp2)
	bodyTypeRegex3, _ := regexp.Compile("(?iU)" + exp3)
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

		typeResult = bodyTypeRegex3.FindStringSubmatch(lineText)
		if len(typeResult) > 1 {
			return strings.ToLower(typeResult[1])
		}
	}
	return ""
}

func (rp *RouteParser) searchForFullPath(name string, lines []string) (result []string) {
	splitName := strings.Split(name, ".")[0]
	exp := "\"(.+/" + splitName + ")\"$"
	regex, _ := regexp.Compile(exp)
	for i := 0; i < len(lines); i++ {
		lineText := lines[i]
		if strings.HasPrefix(lineText, "func") {
			return
		}

		path := regex.FindStringSubmatch(lineText)
		if len(path) > 1 {
			values, err := rp.getFilesFromPath(path[1])
			if err != nil {
				continue
			}

			result = append(result, values...)
		}
	}

	return
}

func (rp *RouteParser) searchCurrentPackage(varType string) (completeVarType string, result []string, err error) {
	lastDotIndex := strings.LastIndex(rp.RelativePath, ".")
	if lastDotIndex == 0 {
		return
	}

	path := rp.RelativePath[:lastDotIndex]
	lastSlashIndex := strings.LastIndex(path, "/")
	completeVarType = strings.Join([]string{path[lastSlashIndex+1:], varType}, ".")
	result, err = rp.getFilesFromPath(path)
	return
}

func (rp *RouteParser) getFilesFromPath(path string) (result []string, err error) {
	goPath := os.Getenv("GOPATH")
	if len(goPath) == 0 {
		goPath = build.Default.GOPATH
	}

	fullPath, err := filepath.Abs(goPath + "/src/" + path)
	if err != nil {
		return
	}

	files, err := ioutil.ReadDir(fullPath)
	if err != nil {
		return
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".go") {
			if !strings.HasSuffix(file.Name(), "_test.go") {
				result = append(result, fullPath+"/"+file.Name())
			}
		}
	}

	return
}
