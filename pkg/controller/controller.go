package controller

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"log"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"os"
	"reflect"
	"strings"
	"time"
)

type DashboardsController struct {
	dashboards     grafana.DashboardsInterface
	clients        kubernetes.Interface
	configmaps     cache.SharedIndexInformer
	errorLogger    *log.Logger
	markerTag      string
	q              workqueue.RateLimitingInterface
	applyPlanFuncs ApplyPlanFuncs
}

const reconcileTask string = "reconcile"

func New(dashboards grafana.DashboardsInterface, clients kubernetes.Interface, configmaps cache.SharedIndexInformer, markerTag string, dryRun bool) *DashboardsController {
	c := &DashboardsController{
		dashboards:  dashboards,
		clients:     clients,
		configmaps:  configmaps,
		errorLogger: log.New(os.Stderr, "", log.LstdFlags),
		markerTag:   markerTag,
		q: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			strings.Join([]string{markerTag, "grafana-dashboards"}, "-"),
		),
	}

	if dryRun {
		c.applyPlanFuncs = ApplyPlanFuncs{
			CreateFunc: func(ctx context.Context, dash *grafana.Dashboard) error {
				slug, _ := dash.Slug()
				log.Printf("created dashboard %v\n", slug)
				return nil
			},
			UpdateFunc: func(ctx context.Context, dash *grafana.Dashboard) error {
				slug, _ := dash.Slug()
				log.Printf("updated dashboard %v\n", slug)
				return nil
			},
			DeleteFunc: func(ctx context.Context, slug string) error {
				log.Printf("deleted dashboard %v\n", slug)
				return nil
			},
		}
	} else {
		c.applyPlanFuncs = ApplyPlanFuncs{
			CreateFunc: dashboards.Import,
			UpdateFunc: func(ctx context.Context, dash *grafana.Dashboard) error {
				slug, err := dash.Slug()
				if err != nil {
					return fmt.Errorf("could not get dashboard slug : %v", err)
				}

				err = dashboards.Delete(ctx, slug)
				if err != nil {
					return fmt.Errorf("could not delete the stale %v dashboard : %v", slug, err)
				}

				err = dashboards.Import(ctx, dash)
				if err != nil {
					return fmt.Errorf("could import the up-to-date %v dashboard : %v", slug, err)
				}
				return nil
			},
			DeleteFunc: dashboards.Delete,
		}
	}

	return c
}

func (c *DashboardsController) Run(ctx context.Context) {
	if !cache.WaitForCacheSync(ctx.Done(), c.configmaps.HasSynced) {
		return
	}

	go func() {
		t := time.NewTicker(60 * time.Second)
		for c.processWorkItem(ctx) {
			select {
			case <-t.C:
			case <-ctx.Done():
				c.q.ShutDown()
				return
			}
		}
	}()

	c.configmaps.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.q.Add(reconcileTask)
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			c.q.Add(reconcileTask)
		},
		DeleteFunc: func(obj interface{}) {
			c.q.Add(reconcileTask)
		},
	})

	<-ctx.Done()
}

func (c *DashboardsController) processWorkItem(ctx context.Context) bool {
	taskObj, shutdown := c.q.Get()
	defer c.q.Done(taskObj)

	if shutdown {
		log.Println("Shutting down the work queue")
		return false
	}

	if taskObj.(string) != reconcileTask {
		c.q.Forget(taskObj)
		return true
	}

	log.Println("reconciling")

	w, err := buildWorldView(ctx, c.errorLogger, c.dashboards, c.configmaps, c.markerTag)
	if err != nil {
		c.errorLogger.Printf("failed to build world view : %v\n", err)
		c.q.AddRateLimited(taskObj)
		return true
	}

	plan := w.PlanChanges()

	log.Println(len(plan.Changes), "changes planned")

	if err = plan.Apply(ctx, c.applyPlanFuncs); err != nil {
		c.errorLogger.Printf("failed to apply plan : %v\n", err)
		c.q.AddRateLimited(taskObj)
	} else {
		log.Println("reconciliation was successful")
	}

	return true
}

func buildWorldView(ctx context.Context, errorLogger *log.Logger, dashboards grafana.DashboardsInterface, configmaps cache.SharedIndexInformer, markerTag string) (*World, error) {
	currentDashboards, err := dashboards.Search(ctx, grafana.DashboardSearchQuery{
		Tags: []string{markerTag},
	})
	if err != nil {
		return nil, fmt.Errorf("could not search for dashboards with marker tag %v : %v", markerTag, err)
	}

	desiredDashboards := []*grafana.Dashboard{}
	for _, cmObj := range configmaps.GetStore().List() {
		cm := cmObj.(*v1.ConfigMap)
		key, err := keyOf(cm)
		if err != nil {
			errorLogger.Printf("failed get key of ConfigMap %v: %v\n", cm, err)
			continue
		}

		desiredDashboard, err := asDashboard(cm)
		if err != nil {
			errorLogger.Printf("could not make a Dashboard out of ConfigMap %v: %v\n", key, err)
			continue
		}

		if err = desiredDashboard.AddTag(markerTag); err != nil {
			errorLogger.Printf("could not add marker tag to Dashboard of ConfigMap %v: %v\n", key, err)
			continue
		}

		desiredDashboards = append(desiredDashboards, desiredDashboard)
	}

	return &World{
		Current: currentDashboards,
		Desired: desiredDashboards,
	}, nil
}

func keyOf(cm *v1.ConfigMap) (string, error) {
	return cache.DeletionHandlingMetaNamespaceKeyFunc(cm)
}

func asDashboard(cm *v1.ConfigMap) (*grafana.Dashboard, error) {
	data := cm.Data
	keys := reflect.ValueOf(data).MapKeys()
	if len(keys) > 0 {
		firstKeyValue := data[keys[0].String()]
		return grafana.NewDashboard([]byte(firstKeyValue))
	}
	key, _ := keyOf(cm)
	return nil, fmt.Errorf("could not get first Data key of ConfigMap %v", key)

}
