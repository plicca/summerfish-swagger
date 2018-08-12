package swagger

import (
	"fmt"
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

func contains(s []NameType, e string) bool {
	for _, a := range s {
		if a.Name == e {
			return true
		}
	}
	return false
}
