package controller

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"log"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/differ"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
	"os"
	"reflect"
	"strings"
	"time"
)

type DashboardsController struct {
	dashboards  grafana.DashboardsInterface
	clients     kubernetes.Interface
	configmaps  cache.SharedIndexInformer
	errorLogger *log.Logger
	markerTag   string
	q           workqueue.RateLimitingInterface
	reconciler  differ.Interface
}

const reconcileTask string = "reconcile"

func New(dashboards grafana.DashboardsInterface, clients kubernetes.Interface, configmaps cache.SharedIndexInformer, markerTag string, dryRun bool) *DashboardsController {
	var reconciler differ.Interface
	if dryRun {
		reconciler = differ.NewFuncsBased(differ.Funcs{
			ListDashboards: func(ctx context.Context) ([]*grafana.DashboardResult, error) {
				return findMarkedDashboards(ctx, dashboards, markerTag)
			},
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
	} else {
		reconciler = differ.NewFuncsBased(differ.Funcs{
			ListDashboards: func(ctx context.Context) ([]*grafana.DashboardResult, error) {
				return findMarkedDashboards(ctx, dashboards, markerTag)
			},
			Create: dashboards.Import,
			Update: dashboards.ImportAndOverwrite,
			Delete: dashboards.Delete,
		})
	}

	return &DashboardsController{
		dashboards:  dashboards,
		clients:     clients,
		configmaps:  configmaps,
		errorLogger: log.New(os.Stderr, "", log.LstdFlags),
		markerTag:   markerTag,
		q: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			strings.Join([]string{markerTag, "grafana-dashboards"}, "-"),
		),
		reconciler: reconciler,
	}
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

	desiredDashboards, err := listDesiredDashboards(ctx, c.errorLogger, c.dashboards, c.configmaps, c.markerTag)
	if err != nil {
		c.errorLogger.Printf("failed to list dashboards declared in the ConfigMaps: %v\n", err)
		c.q.AddRateLimited(taskObj)
		return true
	}

	if err = c.reconciler.Apply(ctx, desiredDashboards); err != nil {
		c.errorLogger.Printf("failed to apply plan : %v\n", err)
		c.q.AddRateLimited(taskObj)
	} else {
		log.Println("reconciliation was successful")
	}

	return true
}

func findMarkedDashboards(ctx context.Context, dashboards grafana.DashboardsInterface, markerTag string) ([]*grafana.DashboardResult, error) {
	return dashboards.Search(ctx, grafana.DashboardSearchQuery{
		Tags: []string{markerTag},
	})
}

func listDesiredDashboards(ctx context.Context, errorLogger *log.Logger, dashboards grafana.DashboardsInterface, configmaps cache.SharedIndexInformer, markerTag string) ([]*grafana.Dashboard, error) {
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

	return desiredDashboards, nil
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
