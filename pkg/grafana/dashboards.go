package grafana

import (
	"context"
	"fmt"
	"net/http"
	"path"
)

const dashboardsPath = "api/dashboards/db"
const importPath = "api/dashboards/import"

type DashboardsClient struct {
	clientBase
}

func (c DashboardsClient) Import(ctx context.Context, dashboard Dashboard) error {
	data, err := dashboard.marshalJSON()
	if err != nil {
		return fmt.Errorf("could not marshal %v: %v", dashboard, err)
	}
	_, err = c.newPostRequest(ctx, importPath, data)
	if err != nil {
		return fmt.Errorf("error while importing dashboard: could create request POST %s with body %v : %v", importPath, dashboard, err)
	}

	/*
		req, err := c.newPostRequest(importPath, data)
		if err != nil {
			return fmt.Errorf("error while importing dashboard: could create request POST %s with body %v : %v", importPath, dashboard, err)
		}

		res, err := c.httpClient.Do(req)

		if err != nil {
			return fmt.Errorf("error while importing dashboard %v: %v", dashboard, err)
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return fmt.Errorf("error while importing dashboard (%s) %v: %v", res.Status, dashboard, err)
		}
	*/

	return nil
}
func (c DashboardsClient) Delete(ctx context.Context, slug string) error {
	uri := path.Join(dashboardsPath, slug)
	//req, err := c.newDeleteRequest(uri)
	_, err := c.newDeleteRequest(ctx, uri)
	if err != nil {
		return fmt.Errorf("error while deleting dashboard %s: could create request %s %s : %v", slug, http.MethodDelete, uri, err)
	}

	/*
		res, err := c.httpClient.Do(req)

		if err != nil {
			return fmt.Errorf("error while deleting dashboard %s: %v", slug, err)
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return fmt.Errorf("error while deleting dashboard (%s) %s: %v", res.Status, slug, err)
		}
	*/

	return nil
}
