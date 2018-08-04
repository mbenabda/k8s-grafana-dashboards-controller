package differ

import (
	"context"
	"fmt"
	"log"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"strings"
)

type Funcs struct {
	Create func(context.Context, *grafana.Dashboard) error
	Update func(context.Context, *grafana.Dashboard) error
	Delete func(context.Context, string) error
}

func NewNoOp() Interface {
	return NewFuncsBased(Funcs{
		Create: func(ctx context.Context, dash *grafana.Dashboard) error {
			slug, _ := dash.Slug()
			log.Printf("created dashboard %v\n", slug)
			return nil
		},
		Update: func(ctx context.Context, dash *grafana.Dashboard) error {
			slug, _ := dash.Slug()
			log.Printf("updated dashboard %v\n", slug)
			return nil
		},
		Delete: func(ctx context.Context, slug string) error {
			log.Printf("deleted dashboard %v\n", slug)
			return nil
		},
	})
}

func New(dashboards grafana.DashboardsInterface) Interface {
	return NewFuncsBased(Funcs{
		Create: func(ctx context.Context, dash *grafana.Dashboard) error {
			slug, err := dash.Slug()
			if err != nil {
				return fmt.Errorf("unable to get slug of dashboard %v", dash)
			}
			err = dashboards.Import(ctx, dash)
			if err != nil {
				return fmt.Errorf("unable to create dashboard %v: %v", slug, err)
			}
			return nil
		},
		Update: func(ctx context.Context, dash *grafana.Dashboard) error {
			slug, err := dash.Slug()
			if err != nil {
				return fmt.Errorf("unable to get slug of dashboard %v", dash)
			}
			err = dashboards.ImportAndOverwrite(ctx, dash)
			if err != nil {
				return fmt.Errorf("unable to update dashboard %v: %v", slug, err)
			}
			return nil
		},
		Delete: func(ctx context.Context, slug string) error {
			err := dashboards.Delete(ctx, slug)
			if err != nil {
				return fmt.Errorf("unable to delete dashboard %v: %v", slug, err)
			}
			return nil
		},
	})
}

func NewFuncsBased(funcs Funcs) Interface {
	return funcsBasedDiffer{funcs}
}

func (this funcsBasedDiffer) Apply(ctx context.Context, current []*grafana.DashboardResult, desired []*grafana.Dashboard) error {
	errors := map[change]error{}
	for _, change := range plan(current, desired) {
		err := change.apply(ctx, this.Funcs)
		if err != nil {
			errors[change] = err
		}
	}

	if len(errors) > 0 {
		prettyErrors := []string{}

		for _, err := range errors {
			prettyErrors = append(prettyErrors, fmt.Sprintf("%v", err))
		}

		return fmt.Errorf(strings.Join(prettyErrors, "\n"))
	}

	return nil
}

type funcsBasedDiffer struct {
	Funcs
}

type change interface {
	apply(context.Context, Funcs) error
}

type createAction struct {
	dashboard *grafana.Dashboard
}
type updateAction struct {
	dashboard *grafana.Dashboard
}
type deleteAction struct {
	slug string
}

func plan(current []*grafana.DashboardResult, desired []*grafana.Dashboard) []change {
	plan := map[string]change{}

	currentSlugs := map[string]bool{}
	for _, dashboard := range current {
		slug, err := dashboard.Slug()
		if err != nil {
			continue
		}
		currentSlugs[slug] = true
		plan[slug] = deleteAction{
			slug: slug,
		}
	}

	for _, dashboard := range desired {
		slug, err := dashboard.Slug()
		if err != nil {
			continue
		}

		if currentSlugs[slug] {
			plan[slug] = updateAction{
				dashboard: dashboard,
			}
		} else {
			plan[slug] = createAction{
				dashboard: dashboard,
			}
		}
	}

	changes := []change{}
	for _, v := range plan {
		changes = append(changes, v)
	}

	return changes
}

func (this createAction) apply(ctx context.Context, funcs Funcs) error {
	return funcs.Create(ctx, this.dashboard)
}

func (this updateAction) apply(ctx context.Context, funcs Funcs) error {
	return funcs.Update(ctx, this.dashboard)
}

func (this deleteAction) apply(ctx context.Context, funcs Funcs) error {
	return funcs.Delete(ctx, this.slug)
}
