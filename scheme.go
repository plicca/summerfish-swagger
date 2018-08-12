package swagger

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type SchemeHolder struct {
	Schemes  []string    `json:"schemes"`
	Host     string      `json:"host"`
	BasePath string      `json:"basePath"`
	Paths    PathsHolder `json:"paths"`
}

func mapRoutesToPaths(routerHolders []RouteHolder) PathsHolder {
	paths := PathsHolder{}
	for _, router := range routerHolders {
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
			parameter := generateInputParameter("body", router.Body.Name, "object")
			parameter.Schema = SchemaParameters{"object", map[string]SchemaParameters{}}
			for _, name := range strings.Split(router.Body.Type, ",") {
				parameter.Schema.Properties[name] = SchemaParameters{Type:"string"}
			}

			parameters = append(parameters, parameter)
		}

		tag := strings.Split(router.Route, "/")[1]
		paths[router.Route][strings.ToLower(router.Methods[0])] = Operation{
			ID: router.Name, Summary: convertCamelCase(router.Name),
			Parameters: parameters,
			Tags: []string{tag},
			Responses: map[string]string{},
		}
	}
	return paths
}

func generateInputParameter(queryType, name, varType string) InputParameter {
	return InputParameter{QueryType: queryType, Type: varType, Name: name, Description: convertCamelCase(name), GoName: name}
}

func convertCamelCase(input string) string {
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
	for _, entry := range runes {
		if len(entry) > 0 {
			entries = append(entries, string(entry))
		}
	}
	return strings.Join(entries, " ")
}
