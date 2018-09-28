package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"
	"time"

	"github.com/integr8ly/integration-controller/pkg/k8s"

	"github.com/integr8ly/integration-controller/pkg/transport"

	"github.com/integr8ly/integration-controller/pkg/consumer"

	"github.com/integr8ly/integration-controller/pkg/fuse"

	"github.com/integr8ly/integration-controller/pkg/enmasse"

	"github.com/integr8ly/integration-controller/pkg/integration"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"

	"github.com/integr8ly/integration-controller/pkg/dispatch"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"github.com/sirupsen/logrus"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func printConfig() {
	logrus.Info("config: insecure requests to services with self signed certs are enabled: ", allowInsecure)
	logrus.Info("config: EnMasse namespace is set to " + enmasseNS)
	logrus.Info("config: fuse namespace is set to " + fuseNS)
	logrus.Info("config: resync interval is set to", resyncPeriod)
	logrus.Info("consig: loglevel ", logLevel)
}

var (
	//various configuration flags
	resyncPeriod        int
	logLevel            string
	fuseNS              string
	enmasseNS           string
	allowInsecure       bool
	enabledIntegrations string
	saToken             string
)

func init() {
	flagset := flag.CommandLine
	flagset.IntVar(&resyncPeriod, "resync", 30, "change the resync period for the watched resources")
	flagset.StringVar(&logLevel, "log-level", logrus.Level.String(logrus.InfoLevel), "Log level to use. Possible values: panic, fatal, error, warn, info, debug")
	flagset.StringVar(&fuseNS, "fuse-namespace", "", "set the namespace the target fuse is running in")
	flagset.StringVar(&enmasseNS, "enmasse-namespace", "enmasse", "set the namespace the target enmasse is running in")
	flagset.BoolVar(&allowInsecure, "allow-insecure", false, "if true invalid certs will be accepted")
	flagset.StringVar(&enabledIntegrations, "watch-for", "all", "comma separated list of services the integration controller should watch resources on. Options: all, enmasse, routes")
	flagset.StringVar(&saToken, "sa-token", "", "pass in an sa token to use. useful for local dev")
}

func main() {
	flag.Parse()
	logLevel, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Errorf("Failed to parse log level: %v", err)
	} else {
		logrus.SetLevel(logLevel)
	}

	sdk.ExposeMetricsPort()

	resource := "integreatly.org/v1alpha1"
	kind := "Integration"
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("Failed to get watch namespace: %v", err)
	}

	if fuseNS == "" {
		fuseNS = namespace
	}

	resync, err := time.ParseDuration(fmt.Sprintf("%vs", resyncPeriod))
	if err != nil {
		panic(err)
	}

	printVersion()
	printConfig()

	k8Client := k8sclient.GetKubeClient()

	routeClient, err := routev1.NewForConfig(k8sclient.GetKubeConfig())
	if err != nil {
		panic(err)
	}

	if allowInsecure {
		logrus.Info("allowing insecure client requests. Accepting self signed certs")
	}

	var (
		httpClient = transport.DefaultHTTPClient(allowInsecure)
		k8sCruder  = &k8s.K8sCrudler{}
		// enmasseService
		enmasseService = enmasse.NewService(k8Client, routeClient, httpClient, enmasseNS)

		// registries
		consumerRegistery    = consumer.Registry{}
		integrationRegistery = integration.Registry{}
	)

	if saToken == "" {
		//read token from file
		data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
		if err != nil {
			panic(err)
		}
		saToken = string(data)
	}

	{
		// add fuse integrations
		c := fuse.NewConsumer(namespace, k8sCruder)
		consumerRegistery.RegisterConsumer(c)
		i := fuse.NewIntegrator(enmasseService, httpClient, namespace, saToken, "integration-controller")
		if err := integrationRegistery.RegisterIntegrator(i); err != nil {
			panic(err)
		}
	}

	integrationReconciler := integration.NewReconciler(integrationRegistery)
	addressSpaceReconciler := enmasse.NewReconciler(consumerRegistery, namespace)
	mainHandler := dispatch.NewHandler(k8Client)
	mainHandler.(*dispatch.Handler).AddHandler(integrationReconciler)
	mainHandler.(*dispatch.Handler).AddHandler(addressSpaceReconciler)
	enabled := strings.Split(enabledIntegrations, ",")
	if isEnabled("enmasse", enabled) {
		sdk.Watch("v1", "ConfigMap", enmasseNS, resync, sdk.WithLabelSelector("type=address-space"))
		logrus.Infof("Watching %s, %s, %s, %d", "", "ConfigMap", namespace, resyncPeriod)
	}
	sdk.Watch(resource, kind, namespace, resync)
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)
	sdk.Handle(mainHandler)
	sdk.Run(context.TODO())
}

func isEnabled(watchService string, enabled []string) bool {
	for _, e := range enabled {
		if watchService == e || e == "all" {
			return true
		}
	}
	return false
}
