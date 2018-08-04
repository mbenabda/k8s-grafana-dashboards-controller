package grafana

import (
	"github.com/gosimple/slug"
	"strings"
)

type Dashboard struct {
	data jsonObj
}
type DashboardResult struct {
	data jsonObj
}

func NewDashboard(body []byte) (*Dashboard, error) {
	j, err := newJsonObj(body)
	if err != nil {
		return nil, err
	}

	return &Dashboard{data: *j}, nil
}

func NewDashboardSearchResults(body []byte) ([]*DashboardResult, error) {
	j, err := newJsonObj(body)
	if err != nil {
		return nil, err
	}
	arr, err := j.asArray()
	if err != nil {
		return nil, err
	}

	dashboards := []*DashboardResult{}

	for _, d := range arr {
		dashboards = append(dashboards, &DashboardResult{data: jsonObj{d}})
	}

	return dashboards, nil
}

func (d Dashboard) Title() (string, error) {
	return d.data.get("dashboard").get("title").String()
}

func (d Dashboard) Slug() (string, error) {
	title, err := d.Title()
	if err != nil {
		return "", err
	}
	return slug.Make(strings.ToLower(title)), nil
}

func (d DashboardResult) Slug() (string, error) {
	uri, err := d.data.get("uri").String()
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(uri, "db/"), nil
}

func (d Dashboard) AddTag(tag string) error {
	if tag == "" {
		return nil
	}

	tagsObj, err := d.data.get("dashboard").get("tags").asArray()
	if err != nil {
		return err
	}
	alreadyHasTag := false
	tags := []string{}
	for _, t := range tagsObj {
		tagStr := t.(string)
		alreadyHasTag = alreadyHasTag || (tagStr == tag)
		tags = append(tags, tagStr)
	}

	if !alreadyHasTag {
		tags = append(tags, tag)
		d.data.get("dashboard").set("tags", tags)
	}

	return nil
}
