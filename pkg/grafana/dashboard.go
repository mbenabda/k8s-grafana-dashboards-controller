package grafana

import (
	"github.com/gosimple/slug"
	"strings"
)

type Dashboard = jsonObj
type DashboardResult = jsonObj

func NewDashboard(body []byte) (Dashboard, error) {
	j, err := newJsonObj(body)
	if err != nil {
		return Dashboard{}, err
	}

	return Dashboard(*j), nil
}

func newDashboardSearchResults(body []byte) ([]DashboardResult, error) {
	j, err := newJsonObj(body)
	if err != nil {
		return nil, err
	}
	arr, err := j.asArray()
	if err != nil {
		return nil, err
	}

	dashboards := []DashboardResult{}

	for _, d := range arr {
		dashboards = append(dashboards, DashboardResult(jsonObj{d}))
	}

	return dashboards, nil
}

func (d Dashboard) Title() (string, error) {
	return d.get("dashboard").get("title").String()
}

func (d Dashboard) Slug() (string, error) {
	title, err := d.Title()
	if err != nil {
		return "", err
	}
	return slug.Make(strings.ToLower(title)), nil
}

func (d Dashboard) AddTag(tag string) error {
	tagsObj, err := d.get("dashboard").get("tags").asArray()
	if err != nil {
		return err
	}
	alreadyHasTag := false
	tags := make([]string, len(tagsObj)+1)
	for _, t := range tagsObj {
		tagStr := t.(string)
		alreadyHasTag = alreadyHasTag || (tagStr == tag)
		tags = append(tags, tagStr)
	}
	if !alreadyHasTag {
		tags = append(tags, tag)
		d.get("dashboard").set("tags", tags)
	}

	return nil
}
