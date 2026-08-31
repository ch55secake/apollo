package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type dashboardJSON struct {
	ID         int64          `json:"id"`
	UID        string         `json:"uid"`
	Title      string         `json:"title"`
	Tags       []string       `json:"tags"`
	Timezone   string         `json:"timezone"`
	Refresh    string         `json:"refresh"`
	Time       TimeRange      `json:"time"`
	Panels     []panelJSON    `json:"panels"`
	Rows       []rowJSON      `json:"rows"`
	Templating templatingJSON `json:"templating"`
}

type panelJSON struct {
	ID            int               `json:"id"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Type          string            `json:"type"`
	GridPos       GridPos           `json:"gridPos"`
	Datasource    json.RawMessage   `json:"datasource"`
	Targets       []json.RawMessage `json:"targets"`
	Panels        []panelJSON       `json:"panels"`
	Options       json.RawMessage   `json:"options"`
	Content       string            `json:"content"`
	MaxDataPoints int               `json:"maxDataPoints"`
	Raw           json.RawMessage   `json:"-"`
}

type rowJSON struct {
	Title  string      `json:"title"`
	Panels []panelJSON `json:"panels"`
}

type templatingJSON struct {
	List []variableJSON `json:"list"`
}

type variableJSON struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Type    string `json:"type"`
	Query   string `json:"query"`
	Current struct {
		Text  json.RawMessage `json:"text"`
		Value json.RawMessage `json:"value"`
	} `json:"current"`
}

func (p *panelJSON) UnmarshalJSON(data []byte) error {
	type panelAlias panelJSON
	var decoded panelAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*p = panelJSON(decoded)
	p.Raw = append(json.RawMessage(nil), data...)
	return nil
}

func Parse(data []byte) (Dashboard, error) {
	raw := bytes.TrimSpace(data)
	if len(raw) == 0 {
		return Dashboard{}, fmt.Errorf("dashboard JSON is empty")
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Dashboard{}, fmt.Errorf("decode dashboard JSON: %w", err)
	}
	if wrapped, ok := envelope["dashboard"]; ok && len(wrapped) > 0 && string(wrapped) != "null" {
		raw = wrapped
	} else if wrapped, ok := envelope["spec"]; ok && len(wrapped) > 0 && string(wrapped) != "null" {
		raw = wrapped
	}

	var parsed dashboardJSON
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Dashboard{}, fmt.Errorf("decode dashboard payload: %w", err)
	}
	if strings.TrimSpace(parsed.Title) == "" {
		parsed.Title = "Untitled dashboard"
	}

	dashboard := Dashboard{
		DashboardSummary: DashboardSummary{
			ID:    strconv.FormatInt(parsed.ID, 10),
			UID:   parsed.UID,
			Title: parsed.Title,
			Tags:  append([]string(nil), parsed.Tags...),
		},
		Timezone: parsed.Timezone,
		Refresh:  parsed.Refresh,
		Time:     parsed.Time,
		Raw:      append(json.RawMessage(nil), raw...),
	}

	for _, panel := range parsed.Panels {
		dashboard.Panels = append(dashboard.Panels, flattenPanel(panel, "")...)
	}
	for _, row := range parsed.Rows {
		for _, panel := range row.Panels {
			dashboard.Panels = append(dashboard.Panels, flattenPanel(panel, row.Title)...)
		}
	}
	for _, variable := range parsed.Templating.List {
		current := variableCurrentValue(variable.Current.Value)
		if current == "" {
			current = variableCurrentValue(variable.Current.Text)
		}
		dashboard.Variables = append(dashboard.Variables, Variable{
			Name: variable.Name, Label: variable.Label, Type: variable.Type,
			Query: variable.Query, Current: current,
		})
	}
	return dashboard, nil
}

func variableCurrentValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	var values []string
	if json.Unmarshal(raw, &values) == nil {
		return strings.Join(values, ",")
	}
	return ""
}

func flattenPanel(raw panelJSON, row string) []Panel {
	panelRow := row
	if raw.Type == "row" && raw.Title != "" {
		panelRow = raw.Title
	}
	if raw.Type == "row" || len(raw.Panels) > 0 {
		if len(raw.Panels) == 0 {
			return []Panel{{ID: raw.ID, Title: raw.Title, Type: raw.Type, Row: panelRow, GridPos: raw.GridPos, Raw: panelRaw(raw)}}
		}
		var panels []Panel
		for _, child := range raw.Panels {
			panels = append(panels, flattenPanel(child, panelRow)...)
		}
		return panels
	}

	datasource := parseDatasource(raw.Datasource)
	panel := Panel{
		ID:            raw.ID,
		Title:         raw.Title,
		Description:   raw.Description,
		Type:          raw.Type,
		GridPos:       raw.GridPos,
		Row:           row,
		Datasource:    datasource,
		MaxDataPoints: raw.MaxDataPoints,
		Options:       append(json.RawMessage(nil), raw.Options...),
		Raw:           panelRaw(raw),
	}
	panel.Text = raw.Content
	if panel.Text == "" && len(raw.Options) > 0 {
		var options struct {
			Content string `json:"content"`
		}
		_ = json.Unmarshal(raw.Options, &options)
		panel.Text = options.Content
	}
	for i, targetRaw := range raw.Targets {
		var targetJSON struct {
			RefID        string          `json:"refId"`
			Expr         string          `json:"expr"`
			LegendFormat string          `json:"legendFormat"`
			Instant      bool            `json:"instant"`
			Range        bool            `json:"range"`
			Datasource   json.RawMessage `json:"datasource"`
		}
		if err := json.Unmarshal(targetRaw, &targetJSON); err != nil {
			continue
		}
		refID := targetJSON.RefID
		if refID == "" {
			refID = strconv.Itoa(i + 1)
		}
		targetDatasource := parseDatasource(targetJSON.Datasource)
		if targetDatasource == (DataSourceRef{}) {
			targetDatasource = datasource
		}
		panel.Targets = append(panel.Targets, Target{
			RefID: refID, Expr: targetJSON.Expr, LegendFormat: targetJSON.LegendFormat,
			Instant: targetJSON.Instant, Range: targetJSON.Range,
			Datasource: targetDatasource, Raw: append(json.RawMessage(nil), targetRaw...),
		})
	}
	return []Panel{panel}
}

func parseDatasource(raw json.RawMessage) DataSourceRef {
	if len(raw) == 0 || string(raw) == "null" {
		return DataSourceRef{}
	}
	var name string
	if json.Unmarshal(raw, &name) == nil {
		return DataSourceRef{Type: name, Name: name}
	}
	var ref struct {
		Type string `json:"type"`
		UID  string `json:"uid"`
		Name string `json:"name"`
	}
	if json.Unmarshal(raw, &ref) != nil {
		return DataSourceRef{}
	}
	return DataSourceRef{Type: ref.Type, UID: ref.UID, Name: ref.Name}
}

func marshalRaw(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}

func panelRaw(panel panelJSON) json.RawMessage {
	if len(panel.Raw) > 0 {
		return append(json.RawMessage(nil), panel.Raw...)
	}
	return marshalRaw(panel)
}
