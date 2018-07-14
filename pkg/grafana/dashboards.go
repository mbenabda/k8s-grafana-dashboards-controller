package grafana

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"path"
)

const dashboardsPath = "api/dashboards/db"
const importPath = "api/dashboards/import"
const searchPath = "api/search"

type DashboardsClient struct {
	clientBase
}

func (c DashboardsClient) Import(ctx context.Context, dashboard *Dashboard) error {
	data, err := dashboard.data.marshalJSON()
	if err != nil {
		return fmt.Errorf("could not marshal %v: %v", dashboard, err)
	}
	_, err = c.newPostRequest(ctx, importPath, data)
	if err != nil {
		slug, _ := dashboard.Slug()
		return fmt.Errorf("error while importing dashboard: %v", slug, err)
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

func (c DashboardsClient) ImportAndOverwrite(ctx context.Context, dashboard *Dashboard) error {
	dashboard.data.set("overwrite", true)
	return c.Import(ctx, dashboard)
}

func (c DashboardsClient) Delete(ctx context.Context, slug string) error {
	uri := path.Join(dashboardsPath, slug)
	_, err := c.newDeleteRequest(ctx, uri)
	if err != nil {
		return fmt.Errorf("error while deleting dashboard %s: %v", slug, err)
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

func (c DashboardsClient) Search(ctx context.Context, query DashboardSearchQuery) ([]*DashboardResult, error) {
	req, err := c.newGetRequest(ctx, searchPath, url.Values{
		"tag": query.Tags,
	})

	if err != nil {
		return nil, fmt.Errorf("could not build dashboards search request with query %v: %v", query, err)
	}
	res, err := c.httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("error while performing dashboard search request with query %v: %v", query, err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		errBody := ""
		if res.Body != nil {
			defer res.Body.Close()
			bodyBytes, _ := ioutil.ReadAll(res.Body)
			errBody = string(bodyBytes)
		}
		return nil, fmt.Errorf("error while searching for dashboard with query %v : %v %v", query, res.Status, errBody)
	}

	defer res.Body.Close()
	bodyBytes, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, fmt.Errorf("could not read search results of dashboards search with query %v: %v", query, err)
	}

	return newDashboardSearchResults(bodyBytes)
}
