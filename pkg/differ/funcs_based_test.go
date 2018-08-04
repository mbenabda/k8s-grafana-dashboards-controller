package differ_test

import (
	"context"
	"fmt"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/differ"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"testing"
)

func TestEmptySets(t *testing.T) {
	changes, err := diff([]*grafana.DashboardResult{}, []*grafana.Dashboard{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("expected no change, got %v : %v", len(changes), changes)
	}
}
func TestIgnoreDesiredDashboardWithoutTitle(t *testing.T) {
	invalidDash, _ := grafana.NewDashboard([]byte(`
		{
			"dashboard": {}
		}
	`))

	changes, err := diff(
		[]*grafana.DashboardResult{},
		[]*grafana.Dashboard{invalidDash})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(changes) != 0 {
		t.Fatalf("invalid desired dashboard should have been ignored, got : %v", changes)
	}
}

func TestShouldCreateMissingDashboard(t *testing.T) {
	dash, _ := grafana.NewDashboard([]byte(`
		{
			"dashboard": {
				"title": "dashboard-1"
			}
		}
	`))

	changes, err := diff(
		[]*grafana.DashboardResult{},
		[]*grafana.Dashboard{dash})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %v : %v", len(changes), changes)
	}

	if changes[0] != fmt.Sprintf("create %v", dash) {
		t.Fatalf("expected missing desired dashboard to be created")
	}
}
func TestShouldUpdateExistingDashboard(t *testing.T) {
	desired, _ := grafana.NewDashboard([]byte(`
		{
			"dashboard": {
				"title": "dashboard-1"
			}
		}
	`))

	currents, _ := grafana.NewDashboardSearchResults([]byte(`
		[
			{
				"uri": "db/dashboard-1"
			}
		]
	`))

	changes, err := diff(
		currents,
		[]*grafana.Dashboard{desired})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %v : %v", len(changes), changes)
	}

	if changes[0] != fmt.Sprintf("update %v", desired) {
		t.Fatalf("expected existing desired dashboard to be updated")
	}
}
func TestShouldDeleteRemovedDashboard(t *testing.T) {
	currents, _ := grafana.NewDashboardSearchResults([]byte(`
		[
			{
				"uri": "db/dashboard-1"
			}
		]
	`))

	changes, err := diff(
		currents,
		[]*grafana.Dashboard{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %v : %v", len(changes), changes)
	}

	if changes[0] != fmt.Sprint("delete dashboard-1") {
		t.Fatalf("expected existing desired dashboard to be updated")
	}
}

func TestIgnoreCurrentDashboardWithoutUri(t *testing.T) {
	currents, _ := grafana.NewDashboardSearchResults([]byte(`
		[
			{}
		]
	`))
	changes, err := diff(
		currents,
		[]*grafana.Dashboard{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(changes) != 0 {
		t.Fatalf("invalid existing dashboard should have been ignored, got : %v", changes)
	}
}

func diff(current []*grafana.DashboardResult, desired []*grafana.Dashboard) ([]string, error) {
	changes := []string{}

	err := differ.NewFuncsBased(differ.Funcs{
		Create: func(ctx context.Context, dash *grafana.Dashboard) error {
			change := fmt.Sprintf("create %v", dash)
			changes = append(changes, change)
			fmt.Printf("change = %v", change)
			return nil
		},
		Update: func(ctx context.Context, dash *grafana.Dashboard) error {
			change := fmt.Sprintf("update %v", dash)
			changes = append(changes, change)
			return nil
		},
		Delete: func(ctx context.Context, slug string) error {
			change := fmt.Sprintf("delete %v", slug)
			changes = append(changes, change)
			fmt.Printf("change = %v", change)
			return nil
		},
	}).Apply(context.Background(), current, desired)

	return changes, err
}
