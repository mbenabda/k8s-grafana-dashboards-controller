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
	"time"

	"reflect"
)

type DashboardsController struct {
	dashboards  grafana.DashboardsInterface
	clients     kubernetes.Interface
	configmaps  cache.SharedIndexInformer
	errorLogger *log.Logger
	markerTag   string
	q           workqueue.RateLimitingInterface
}

const reconcileTask string = "reconcile"

func New(dashboards grafana.DashboardsInterface, clients kubernetes.Interface, configmaps cache.SharedIndexInformer, markerTag string) *DashboardsController {
	return &DashboardsController{
		dashboards:  dashboards,
		clients:     clients,
		configmaps:  configmaps,
		errorLogger: log.New(os.Stderr, "", log.LstdFlags),
		markerTag:   markerTag,
		q: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			markerTag+"-grafana-dashboards",
		),
	}
}

func keyOf(cm *v1.ConfigMap) (string, error) {
	return cache.DeletionHandlingMetaNamespaceKeyFunc(cm)
}

func (c *DashboardsController) asDashboard(cm *v1.ConfigMap) (grafana.Dashboard, error) {
	data := cm.Data
	keys := reflect.ValueOf(data).MapKeys()
	if len(keys) > 0 {
		firstKeyValue := data[keys[0].String()]
		dashboard, err := grafana.NewDashboard([]byte(firstKeyValue))
		dashboard.AddTag(c.markerTag)
		return dashboard, err
	}
	key, _ := keyOf(cm)
	return grafana.Dashboard{}, fmt.Errorf("could not get first Data key of ConfigMap %v", key)

}

func (c *DashboardsController) Run(ctx context.Context) {
	if !cache.WaitForCacheSync(ctx.Done(), c.configmaps.HasSynced) {
		return
	}

	go func() { c.runWorker(ctx) }()

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

func (c *DashboardsController) runWorker(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	for c.processWorkItem(ctx) {
		select {
		case <-t.C:
		case <-ctx.Done():
			break
		}
	}
	c.q.ShutDown()
}

func (c *DashboardsController) processWorkItem(ctx context.Context) bool {
	keyObj, _ /* shutdown */ := c.q.Get()
	defer c.q.Done(keyObj)

	c.reconcile(ctx)

	return true
}

func (c *DashboardsController) reconcile(ctx context.Context) {
	log.Println("reconciling")

	results, err := c.dashboards.Search(ctx, grafana.DashboardSearchQuery{
		Tags: []string{c.markerTag},
	})

	if err != nil {
		c.errorLogger.Printf("could not search for dashboards with market tag %v : %v\n", c.markerTag, err)
		return
	}

	log.Printf("found %v dashboards with the marker tag", len(results))
}

func (c *DashboardsController) createDashboard(ctx context.Context, cm *v1.ConfigMap) error {
	dashboard, err := c.asDashboard(cm)
	if err != nil {
		return fmt.Errorf("could not parse ConfigMap as a Dashboard : %v", err)
	}

	if err := c.dashboards.Import(ctx, dashboard); err != nil {
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *DashboardsController) deleteDashboard(ctx context.Context, cm *v1.ConfigMap) error {
	dashboard, err := c.asDashboard(cm)
	if err != nil {
		return fmt.Errorf("could not parse ConfigMap as a Dashboard : %v", err)
	}

	slug, err := dashboard.Slug()
	if err != nil {
		return fmt.Errorf("could not build slug of Dashboard defined in ConfigMap : %v", err)
	}

	if err := c.dashboards.Delete(ctx, slug); err != nil {
		if err != nil {
			return err
		}
	}
	return nil
}
