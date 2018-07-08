package grafana

import (
	"github.com/gosimple/slug"
	"strings"
)

type Dashboard = jsonObj

func NewDashboard(body []byte) (Dashboard, error) {
	j, err := newJsonObj(body)
	if err != nil {
		return Dashboard{}, err
	}

	return Dashboard(*j), nil
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
