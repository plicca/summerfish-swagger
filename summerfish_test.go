package summerfish

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestProcessSourceFiles(t *testing.T) {
	type args struct {
		lines []string
	}
	tests := []struct {
		name   string
		args   args
		result []string
	}{
		{
			"Process Url Query Parameters",
			args{[]string{
				"activity := transport.StoryActivity{",
				"UserID:       r.Header.Get(component.AuthHeader),",
				"StoryID:      r.URL.Query().Get(\"storyId\"),",
				"ActivityType: r.URL.Query().Get(\"type\"),",
				" }",
				"}",
			},
			},
			[]string{"storyId", "type"},
		},
	}

	routeParser := RouteParser{RelativePath: "."}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := routeParser.processSourceFiles(tt.args.lines)
			for _, entry := range tt.result {
				if !contains(result.Query, entry) {
					t.Fatal(tt.name, tt.result, result.Query)
					return
				}
			}
		})
	}
}

func TestProcessBodyVars(t *testing.T) {
	type args struct {
		lines []string
	}
	tests := []struct {
		name   string
		args   args
		result []string
	}{
		{
			"Process Body Parameters",
			args{[]string{
				"import (",
				"\"encoding/json\"",
				"\"net/http\"",
				"\"strconv\"",
				"\"cmd/component\"",
				"\"cmd/model/nosql\"",
				"\"cmd/model/transport\"",
				"\"cmd/service\"",
				")",
				"func UpdateActivity(w http.ResponseWriter, r *http.Request) {",
				"	userID := r.Header.Get(authHeader)",
				"",
				"	var activity transport.StoryActivity",
				"	err := json.NewDecoder(r.Body).Decode(&activity)",
				"	if component.ControllerError(w, err, component.ErrInvalidParams) {",
				"		return",
				"	}",
				"",
				"	activity.UserID = userID",
				"",
				"	updatedActivity, err := service.UpdateActivity(activity)",
				"	if component.ControllerError(w, err, nil) {",
				"		return",
				"	}",
				"",
				"	component.ReturnAsJsonResponse(w, updatedActivity)",
				"}",
			},
			},
			[]string{},
		},
	}

	routeParser := RouteParser{RelativePath: "."}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := routeParser.processSourceFiles(tt.args.lines)
			fmt.Println(result)
		})
	}
}

func TestProcessArrayVars(t *testing.T) {
	type args struct {
		lines NameType
	}
	tests := []struct {
		name   string
		args   args
		result string
	}{
		{
			"Process Body Parameters",
			args{NameType{
				Name:    "Client",
				IsArray: false,
				Children: []NameType{
					{Name: "prop1", Type: "type1", IsArray: true, Children: []NameType{{Name: "prop1_1", Type: "string"}, {Name: "prop1_2", Type: "string"}, {Name: "prop1_3", Type: "number"}}},
					{Name: "prop2", Type: "number"},
					{Name: "prop3", Type: "type2", IsArray: true, Children: []NameType{{Name: "prop3_1", Type: "string"}, {Name: "prop3_2", Type: "string"}, {Name: "prop3_3", Type: "number"}, {Name: "prop3_4", Type: "string"}}},
					{Name: "prop4", Type: "string"},
					{Name: "prop5", Type: "string"},
					{Name: "prop6", Type: "string", IsArray: true},
					{Name: "prop7", Type: "type3", IsArray: true, Children: []NameType{{Name: "prop7_1", Type: "string"}, {Name: "prop7_2", Type: "string"}, {Name: "prop7_3", Type: "number"}}},
					{Name: "prop8", Type: "type2", IsArray: false, Children: []NameType{{Name: "prop8_1", Type: "string"}, {Name: "prop8_2", Type: "string"}, {Name: "prop8_3", Type: "number"}, {Name: "prop8_4", Type: "string"}}},
				},
			},
			},
			"\"schema\":{\"type\":\"object\",\"properties\":{\"prop1\":{\"type\":\"array\",\"items\":{\"type\":\"object\",\"properties\":{\"prop1_1\":{\"type\":\"string\"},\"prop1_2\":{\"type\":\"string\"},\"prop1_3\":{\"type\":\"number\"}}}},\"prop2\":{\"type\":\"number\"},\"prop3\":{\"type\":\"array\",\"items\":{\"type\":\"object\",\"properties\":{\"prop3_1\":{\"type\":\"string\"},\"prop3_2\":{\"type\":\"string\"},\"prop3_3\":{\"type\":\"number\"},\"prop3_4\":{\"type\":\"string\"}}}},\"prop4\":{\"type\":\"string\"},\"prop5\":{\"type\":\"string\"},\"prop6\":{\"type\":\"array\",\"items\":{\"type\":\"string\"}},\"prop7\":{\"type\":\"array\",\"items\":{\"type\":\"object\",\"properties\":{\"prop7_1\":{\"type\":\"string\"},\"prop7_2\":{\"type\":\"string\"},\"prop7_3\":{\"type\":\"number\"}}}},\"prop8\":{\"type\":\"object\",\"properties\":{\"prop8_1\":{\"type\":\"string\"},\"prop8_2\":{\"type\":\"string\"},\"prop8_3\":{\"type\":\"number\"},\"prop8_4\":{\"type\":\"string\"}}}}}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapBodyRoute(tt.args.lines)
			/*			fmt.Printf("%+v\n", result)*/

			op := Operation{
				ID:         "Something",
				Summary:    convertCamelCase("Something"),
				Parameters: []InputParameter{result},
				Tags:       []string{"TAG"},
				Responses:  map[string]string{},
			}

			encoded, err := json.Marshal(op)
			if err != nil {
				return
			}
			fmt.Println(strings.Split(string(encoded), "schema")[1])
			fmt.Println(strings.Split(tt.result, "schema")[1])
			if !strings.Contains(string(encoded), tt.result) {
				t.Fatal(string(encoded), tt.result)
			}
			fmt.Println(string(encoded))
		})
	}
}

func contains(s []NameType, e string) bool {
	for _, a := range s {
		if a.Name == e {
			return true
		}
	}
	return false
}
