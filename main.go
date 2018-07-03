package main

import (
	"context"
	"fmt"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"os"

	"log"
	"net/url"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	controller "mbenabda.com/k8s-grafana-dashboards-controller/controller"
	"mbenabda.com/k8s-grafana-dashboards-controller/grafana"
)

type FilterOptions struct {
	Namespace     string
	LabelSelector labels.Selector
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

	kingpin.CommandLine.Help = "Kubernetes controller that reconciles grafana configuration with dashboards defined as configmaps"

	kingpin.Flag("grafana-url", "url to grafana").
		Envar("GRAFANA_API_URL").
		Required().
		URLVar(&grafanaOptions.URL)

	kingpin.Flag("grafana-api-key", "grafana API Key").
		Envar("GRAFANA_API_KEY").
		StringVar(&grafanaOptions.APIKey)

	kingpin.Flag("grafana-user", "grafana User name (Basic Auth). Required unless using an API key").
		Envar("GRAFANA_BASIC_AUTH_USERNAME").
		StringVar(&grafanaOptions.Username)

	kingpin.Flag("grafana-password", "grafana User password (Basic Auth). Required unless using an API key").
		Envar("GRAFANA_BASIC_AUTH_PASSWORD").
		StringVar(&grafanaOptions.Password)

	kingpin.Flag("watch-namespace", "namespace to wath for Configmaps. Defaults to the namespace the controller runs into").
		StringVar(&filterOptions.Namespace)

	selector := SelectorValueHolder(kingpin.
		Flag("selector", "configmap labels selector").
		PlaceHolder("label1=value1,label2=value2").
		Envar("CONFIGMAP_SELECTOR"))

	kingpin.Flag("kubeconfig", "path to a kubernetes config file defining a \"current\" context.").
		ExistingFileVar(&kubeconfig)

	kingpin.Parse()

	filterOptions.LabelSelector = selector.Value

	if err := runWith(kubeconfig, grafanaOptions, filterOptions); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n\n", err)
		kingpin.Usage()
		os.Exit(1)
	}
}

func runWith(kubeconfig string, grafanaOpts *GrafanaOptions, filterOpts *FilterOptions) error {
	grafana, err := buildGrafanaClient(grafanaOpts)
	if err != nil {
		return fmt.Errorf("could not build a grafana client : %v", err)
	}

	k8sConfig, err := buildK8sConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("could not build kubernetes configuration : %v", err)
	}

	return run(grafana, k8sConfig, filterOpts.Namespace, filterOpts.LabelSelector)
}

func run(grafana grafana.Interface, k8sConfig *rest.Config, namespace string, labelsSelector labels.Selector) error {
	controller := controller.New(
		grafana.Dashboards(),
		k8sConfig,
		namespace,
		labelsSelector,
	)

	sig := make(chan os.Signal)
	defer close(sig)

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	errs := make(chan error)
	defer close(errs)

	ctx, cancel := context.WithCancel(context.Background())
	wg, ctx := errgroup.WithContext(ctx)
	defer cancel()

	wg.Go(func() error { return controller.Run(ctx) })

	go func() {
		if err := wg.Wait(); err != nil {
			errs <- err
			return
		}
	}()

	select {
	case s := <-sig:
		log.Printf("received %v signal. Shutting down\n", s)
	case err := <-errs:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		log.Println("work done. exiting")
	}

	return nil
}

func buildGrafanaClient(opts *GrafanaOptions) (grafana.Interface, error) {
	if opts.URL == nil {
		return nil, fmt.Errorf("an url is required")
	}

	if opts.Username != "" || opts.Password != "" {
		return grafana.NewWithUserCredentials(opts.URL, opts.Username, opts.Password)
	}

	return grafana.NewWithApiKey(opts.URL, opts.APIKey)
}

func buildK8sConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		log.Printf("reading the kubeconfig from %s\n", kubeconfig)
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	log.Println("resolving in-cluster configuration")
	return rest.InClusterConfig()
}
