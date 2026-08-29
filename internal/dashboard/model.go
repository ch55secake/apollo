package dashboard

import (
	"context"
	"encoding/json"
	"strings"
)

type Source interface {
	List(ctx context.Context) ([]DashboardSummary, error)
	Get(ctx context.Context, id string) (Dashboard, error)
}

type DashboardSummary struct {
	ID        string
	UID       string
	Title     string
	Tags      []string
	URL       string
	FolderUID string
	Starred   bool
	Source    string
}

type Dashboard struct {
	DashboardSummary
	Timezone  string
	Refresh   string
	Time      TimeRange
	Panels    []Panel
	Variables []Variable
	Raw       json.RawMessage
}

type TimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type GridPos struct {
	H int `json:"h"`
	W int `json:"w"`
	X int `json:"x"`
	Y int `json:"y"`
}

type Panel struct {
	ID            int
	Title         string
	Description   string
	Type          string
	GridPos       GridPos
	Row           string
	Datasource    DataSourceRef
	Targets       []Target
	Text          string
	MaxDataPoints int
	Options       json.RawMessage
	Raw           json.RawMessage
}

type Target struct {
	RefID        string
	Expr         string
	LegendFormat string
	Instant      bool
	Range        bool
	Datasource   DataSourceRef
	Raw          json.RawMessage
}

type DataSourceRef struct {
	Type string
	UID  string
	Name string
}

func (r DataSourceRef) IsPrometheus() bool {
	value := strings.ToLower(strings.TrimSpace(r.Type + " " + r.Name))
	return value == "" || strings.Contains(value, "prometheus") || value == "$datasource"
}

type Variable struct {
	Name    string
	Label   string
	Type    string
	Query   string
	Current string
}

func ExpandQuery(query string, variables []Variable) string {
	result := query
	for _, variable := range variables {
		if variable.Name == "" || variable.Current == "" {
			continue
		}
		result = strings.ReplaceAll(result, "${"+variable.Name+"}", variable.Current)
		result = strings.ReplaceAll(result, "$"+variable.Name, variable.Current)
		result = strings.ReplaceAll(result, "[["+variable.Name+"]]", variable.Current)
	}
	return result
}
