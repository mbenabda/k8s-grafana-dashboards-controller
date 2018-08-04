package main

import (
	"context"
	"fmt"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/differ"
	"time"

	"os"

	"log"
	"net/url"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	controller "mbenabda.com/k8s-grafana-dashboards-controller/pkg/controller"
	"mbenabda.com/k8s-grafana-dashboards-controller/pkg/grafana"
)

type FilterOptions struct {
	Namespace     string
	LabelSelector labels.Selector
	MarkerTag     string
}

type GrafanaOptions struct {
	URL      *url.URL
	APIKey   string
	Username string
	Password string
}

type LabelSelectorValueHolder struct {
	Value labels.Selector
}

func (s *LabelSelectorValueHolder) Set(value string) error {
	selector, err := labels.Parse(value)
	if err != nil {
		return fmt.Errorf("'%s' is not a valid labels selector: %v", value, err)
	}
	s.Value = selector
	return nil
}

func (s *LabelSelectorValueHolder) String() string {
	return s.Value.String()
}

func SelectorValueHolder(s kingpin.Settings) (target *LabelSelectorValueHolder) {
	target = &LabelSelectorValueHolder{}
	s.SetValue((*LabelSelectorValueHolder)(target))
	return
}

func main() {
	grafanaOptions := &GrafanaOptions{}
	filterOptions := &FilterOptions{}
	var kubeconfig string
	var dryRun bool

	kingpin.CommandLine.Help = "Kubernetes controller that reconciles grafana configuration with dashboards defined as configmaps"

	kingpin.Flag("dry-run", "do not perform write operations against grafana api").
		Envar("DRY_RUN").
		BoolVar(&dryRun)

	kingpin.Flag("grafana-url", "url to grafana").
		Envar("GRAFANA_URL").
		Required().
		URLVar(&grafanaOptions.URL)

	kingpin.Flag("grafana-api-key", "grafana API Key").
		Envar("GRAFANA_API_KEY").
		StringVar(&grafanaOptions.APIKey)

	kingpin.Flag("grafana-user", "grafana User name (Basic Auth). Required unless using an API key").
		Envar("GRAFANA_BASIC_AUTH_USERNAME").
		StringVar(&grafanaOptions.Username)

	kingpin.Flag("grafana-password", "grafana User password (Basic Auth)").
		Envar("GRAFANA_BASIC_AUTH_PASSWORD").
		StringVar(&grafanaOptions.Password)

	kingpin.Flag("marker-tag", "unique tag value to be used as a marker for dashboards managed by this instance of the controller").
		Envar("MARKER_TAG").
		PlaceHolder("managed").
		StringVar(&filterOptions.MarkerTag)

	kingpin.Flag("watch-namespace", "namespace to wath for Configmaps. defaults to all namespaces").
		Envar("WATCH_NAMESPACE").
		Default(v1.NamespaceAll).
		StringVar(&filterOptions.Namespace)

	selector := SelectorValueHolder(kingpin.
		Flag("selector", "configmap labels selector").
		Envar("CONFIGMAP_SELECTOR").
		Default(labels.Everything().String()).
		PlaceHolder("role=grafana-dashboard"))

	kingpin.Flag("kubeconfig", "path to a kubernetes config file defining a \"current\" context. Do not specify when running in cluster").
		ExistingFileVar(&kubeconfig)

	kingpin.Parse()

	filterOptions.LabelSelector = selector.Value

	grafana, err := buildGrafanaClient(grafanaOptions)
	if err != nil {
		errorLogger().Printf("could not build a grafana client : %v\n\n", err)
		kingpin.Usage()
		os.Exit(1)
	}

	restConfig, err := buildK8sConfig(kubeconfig)
	if err != nil {
		errorLogger().Printf("could not build kubernetes configuration : %v\n\n", err)
		kingpin.Usage()
		os.Exit(1)
	}

	clients, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		errorLogger().Printf("could not create kubernetes client : %v\n", err)
		os.Exit(1)
	}

	log.Println("[ dry-run =", dryRun, "]", "running against", grafanaOptions.URL, "with", *filterOptions)
	var reconciler differ.DashboardsChangesApplyFuncs
	if dryRun {
		reconciler = differ.NoOpPlanApplyFuncs
	} else {
		reconciler = differ.NewClientBasedPlanApplyFuncs(grafana.Dashboards())
	}

	run(grafana,
		clients,
		filterOptions,
		differ.NewPlanner(),
		reconciler)
}

func run(grafana grafana.Interface, clients kubernetes.Interface, filterOptions *FilterOptions, dashboardChangesPlanner differ.DashboardsChangesPlanner, reconciler differ.DashboardsChangesApplyFuncs) {
	configmaps := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = filterOptions.LabelSelector.String()
				return clients.CoreV1().ConfigMaps(filterOptions.Namespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = filterOptions.LabelSelector.String()
				return clients.CoreV1().ConfigMaps(filterOptions.Namespace).Watch(options)
			},
		},
		&v1.ConfigMap{},
		1*time.Hour,
		cache.Indexers{},
	)

	sig := make(chan os.Signal)
	defer close(sig)

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	wg, ctx := errgroup.WithContext(ctx)
	defer cancel()

	wg.Go(func() error {
		configmaps.Run(ctx.Done())
		return nil
	})

	wg.Go(func() error {
		controller.New(
			grafana.Dashboards(),
			configmaps,
			filterOptions.MarkerTag,
			dashboardChangesPlanner,
			reconciler,
		).Run(ctx)
		return nil
	})

	select {
	case s := <-sig:
		log.Printf("received %v signal. Shutting down\n", s)
		cancel()
	case <-ctx.Done():
	}

	wg.Wait()
}

func buildGrafanaClient(opts *GrafanaOptions) (grafana.Interface, error) {
	if opts.URL == nil {
		return nil, fmt.Errorf("an url is required")
	}

	if opts.Username == "" && opts.Password == "" {
		return grafana.NewWithApiKey(opts.URL, opts.APIKey)
	}

	return grafana.NewWithUserCredentials(opts.URL, opts.Username, opts.Password)
}

func buildK8sConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	return rest.InClusterConfig()
}

func errorLogger() *log.Logger {
	return log.New(os.Stderr, "", log.LstdFlags)
}
