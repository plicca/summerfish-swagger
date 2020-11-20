package summerfish

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type SchemeHolder struct {
	SwaggerVersion string            `json:"swagger" yaml:"swagger"`
	Information    SchemeInformation `json:"info" yaml:"info"`
	Host           string            `json:"host"`
	BasePath       string            `json:"basePath" yaml:"basePath"`
	Schemes        []string          `json:"schemes"`
	Paths          PathsHolder       `json:"paths"`
}

type SchemeInformation struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	Title   string `json:"title,omitempty" yaml:"title,omitempty"`
}

var jsonMapping = map[string]string{
	"bool":       "boolean",
	"string":     "string",
	"int":        "number",
	"int8":       "number",
	"int16":      "number",
	"int32":      "number",
	"int64":      "number",
	"uint":       "number",
	"uint8":      "number",
	"uint16":     "number",
	"uint32":     "number",
	"uint64":     "number",
	"uintptr":    "number",
	"byte":       "number",
	"rune":       "number",
	"float32":    "number",
	"float64":    "number",
	"complex64":  "number",
	"complex128": "number",
}

var link = regexp.MustCompile("(^[A-Za-z])|_([A-Za-z])")

func mapRoutesToPaths(routerHolders []RouteHolder) PathsHolder {
	paths := PathsHolder{}
	for i, router := range routerHolders {
		if len(router.Methods) == 0 {
			continue
		}

		if _, ok := paths[router.Route]; !ok {
			paths[router.Route] = Method{}
		}

		//Must be initialized like this so that empty converts to json properly
		parameters := []InputParameter{}
		for _, entry := range router.Query {
			parameters = append(parameters, generateInputParameter("query", entry.Name, entry.Type))
		}

		for _, entry := range router.Path {
			parameters = append(parameters, generateInputParameter("path", entry.Name, entry.Type))
		}

		if len(router.Body.Name) > 0 {
			parameters = append(parameters, mapBodyRoute(router.Body))
		}

		hasFormData := false
		for _, entry := range router.FormData {
			parameters = append(parameters, generateInputParameter("formData", entry.Name, entry.Type))
			hasFormData = true
		}

		tag := strings.Split(router.Route, "/")[1]
		operation := Operation{
			ID:         fmt.Sprintf("%s_%d", router.Name, i),
			Summary:    convertFromCamelCase(router.Name),
			Parameters: parameters,
			Tags:       []string{tag},
			Responses:  map[string]OperationResponse{"200": OperationResponse{Description: "successful operation"}},
		}

		if hasFormData {
			operation.Consumes = []string{"multipart/form-data"}
		}

		paths[router.Route][strings.ToLower(router.Methods[0])] = operation
	}

	return paths
}

func mapBodyRoute(bodyField NameType) (result InputParameter) {
	result = generateInputParameter("body", bodyField.Name, "object")
	//result.Schema = mapInternalParameters(bodyField)
	return
}

func mapInternalParameters(bodyField NameType) SchemaParameters {
	props := make(map[string]SchemaParameters)
	for _, param := range bodyField.Children {
		if len(param.Children) > 0 {
			props[param.Name] = mapInternalParameters(param)

		} else {
			mappedParamType, ok := jsonMapping[param.Type]
			if !ok {
				mappedParamType = param.Type
			}

			if param.IsArray {
				props[param.Name] = SchemaParameters{Type: "array", Items: &SchemaParameters{Type: mappedParamType}}
			} else {
				props[param.Name] = SchemaParameters{Type: mappedParamType}
			}
		}
	}

	if bodyField.IsArray {
		items := &SchemaParameters{Type: "object", Properties: props}
		return SchemaParameters{Type: "array", Items: items}
	}

	return SchemaParameters{Type: "object", Properties: props}
}

func generateInputParameter(queryType, name, varType string) InputParameter {
	ip := InputParameter{
		QueryType:   queryType,
		Type:        varType,
		Name:        name,
		Description: name,
	}

	//convert from snake case since camelcase is needed for the next step
	if strings.Contains(name, "_") {
		ip.Description = convertToCamelCase(ip.Description)
	}

	ip.Description = convertFromCamelCase(ip.Description)
	return ip
}

func convertToCamelCase(str string) string {
	return link.ReplaceAllStringFunc(str, func(s string) string {
		return strings.ToUpper(strings.Replace(s, "_", "", -1))
	})
}

func convertFromCamelCase(input string) string {
	if !utf8.ValidString(input) {
		return input
	}

	var entries []string
	var runes [][]rune
	lastClass := 0
	class := 0

	// split into fields based on class of unicode character
	for _, letter := range input {
		switch true {
		case unicode.IsLower(letter):
			class = 1
		case unicode.IsUpper(letter):
			class = 2
		case unicode.IsDigit(letter):
			class = 3
		default:
			class = 4
		}

		if class == lastClass {
			runes[len(runes)-1] = append(runes[len(runes)-1], letter)
		} else {
			runes = append(runes, []rune{letter})
		}

		lastClass = class
	}

	for i := 0; i < len(runes)-1; i++ {
		if unicode.IsUpper(runes[i][0]) && unicode.IsLower(runes[i+1][0]) {
			runes[i+1] = append([]rune{runes[i][len(runes[i])-1]}, runes[i+1]...)
			runes[i] = runes[i][:len(runes[i])-1]
		}
	}

	for i, entry := range runes {
		if len(entry) == 0 {
			continue
		}

		if i == 0 && unicode.IsLower(entry[0]) {
			entry[0] -= 32
		}

		entries = append(entries, string(entry))
	}

	return strings.Join(entries, " ")
}
