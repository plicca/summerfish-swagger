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
	Path     []NameType
	Query    []NameType
	Body     NameType
	FormData []NameType
	Route    string
	Methods  []string
	Name     string
}

type NameType struct {
	Name     string
	Type     string
	IsArray  bool
	Children []NameType
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
	bodyFormFileRegex, _ := regexp.Compile("r\\.FormFile\\(\"(.+)\"\\)")
	bodyFormValueRegex, _ := regexp.Compile("r\\.FormValue\\(\"(.+)\"\\)")

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
		candidateSourceFiles, err = rp.searchForFullPath(varType, lines)
	}
	if err != nil {
		return NameType{Name: name, Type: ""}
	}

	if len(candidateSourceFiles) > 0 {
		return rp.searchForStruct(varType, "", candidateSourceFiles, false)
	}

	return NameType{Name: name, Type: ""}
}

func (rp *RouteParser) searchForStruct(name string, childrenNameFromParent string, paths []string, isArray bool) (result NameType) {
	structInfo := strings.Split(name, ".")
	structPackage := structInfo[0]
	structName := structInfo[1]
	comp := "type " + structName + " struct"
	exp := "^\\s*(.+)\\b\\s+(.+)\\b(\\s+`(.+)`)?$"
	bodyTypeRegex, _ := regexp.Compile(exp)

	if len(childrenNameFromParent) > 0 {
		result.Name = childrenNameFromParent
	} else {
		result.Name = structName
	}

	result.IsArray = isArray

	for _, path := range paths {
		isFound := false
		var file *os.File
		file, _ = os.Open(path)

		commentSection := false

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lineText := scanner.Text()

			lineText, commentSection = cleanCommentSection(lineText, commentSection)

			if isFound {
				if lineText == "}" {
					file.Close()
					return
				}

				typeResult := bodyTypeRegex.FindStringSubmatch(lineText)
				if len(typeResult) > 1 {
					result.Children = append(result.Children, rp.findNativeType(structPackage, typeResult[1], typeResult[2], typeResult[3], paths))
				}
			} else if strings.HasPrefix(lineText, comp) {
				isFound = true
			}
		}
		file.Close()
	}
	return
}

func (rp *RouteParser) findNativeType(structPackage string, varName, varType, varTags string, paths []string) (output NameType) {
	exp := "json:\"(.+)\""
	jsonNameRegex, _ := regexp.Compile(exp)

	if len(varTags) > 0 {
		results := jsonNameRegex.FindStringSubmatch(varTags)
		if len(results) > 1 {
			splitResult := strings.Split(results[1], ",")
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
		return NameType{varName, varType, isArray, nil}
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

func (rp *RouteParser) searchForFullPath(name string, lines []string) (result []string, err error) {
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
			var values []string
			values, err = rp.getFilesFromPath(path[1])
			result = append(result, values...)
		}
	}
	return
}

func (rp *RouteParser) searchCurrentPackage(varType string) (completeVarType string, result []string, err error) {
	path := strings.Split(rp.RelativePath, ".")[0]
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

	var fullPath string
	fullPath, err = filepath.Abs(goPath + "/src/" + path)
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
			if !strings.HasSuffix(file.Name(), "_test.go") {
				result = append(result, fullPath+"/"+file.Name())
			}
		}
	}
	return
}
