package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/integr8ly/integration-controller/pkg/k8s"

	"github.com/integr8ly/integration-controller/pkg/transport"

	"github.com/integr8ly/integration-controller/pkg/consumer"

	"github.com/integr8ly/integration-controller/pkg/fuse"

	"github.com/integr8ly/integration-controller/pkg/enmasse"

	"github.com/integr8ly/integration-controller/pkg/integration"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"

	_ "github.com/integr8ly/integration-controller/pkg/openshift"
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
	logrus.Info("config: loglevel ", logLevel)
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
	// env var
	envUserNamespaces string
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

	envUserNamespaces = os.Getenv("USER_NAMESPACES")

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
		additional           = strings.Split(envUserNamespaces, ",")
	)

	if saToken == "" {
		//read token from file
		data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
		if err != nil {
			panic(err)
		}
		saToken = string(data)
	}

	// add fuse integrations
	{
		fuseConnectionCruder := fuse.NewConnectionCruder(httpClient, namespace, saToken, "integration-controller")
		i := fuse.NewIntegrator(enmasseService, fuseConnectionCruder)
		if err := integrationRegistery.RegisterIntegrator(i); err != nil {
			panic(err)
		}
		httpI := fuse.NewHTTPIntegrator(k8sCruder, fuseConnectionCruder)
		if err := integrationRegistery.RegisterIntegrator(httpI); err != nil {
			panic(err)
		}
		ac := fuse.NewAddressSpaceConsumer(namespace, k8sCruder)
		consumerRegistery.RegisterConsumer(ac)
		rc := fuse.NewServiceRouteConsumer(namespace, routeClient, k8sCruder)
		consumerRegistery.RegisterConsumer(rc)

	}

	integrationHandler := integration.NewReconciler(integrationRegistery)
	consumerHandler := consumer.NewHandler(consumerRegistery, integrationHandler, namespace)
	enabled := strings.Split(enabledIntegrations, ",")

	if isEnabled("enmasse", enabled) {
		// TODO post 0.23.0 move to watching just this namespace
		sdk.Watch("v1", "ConfigMap", enmasseNS, resync, sdk.WithLabelSelector("type=address-space"))
		logrus.Infof("EnMasse integrations enabled. Watching %s, %s, %s, %d", "", "ConfigMap", namespace, resyncPeriod)
	}

	sdk.Watch(resource, kind, namespace, resync)
	logrus.Infof("Watching %s, %s, %sx, %d", resource, kind, namespace, resyncPeriod)

	//refactor seems redundant
	userNSResoures := []schema.GroupVersionKind{
		{
			Group:   "route.openshift.io",
			Version: "v1",
			Kind:    "Route",
		},
		{
			Group:   "",
			Version: "v1",
			Kind:    "Service",
		},
	}

	for _, ans := range additional {
		if ans == "" {
			continue
		}
		for _, gvk := range userNSResoures {
			sdk.Watch(gvk.GroupVersion().String(), gvk.Kind, ans, resync)
			logrus.Infof("Watching additional ns for %s, %s, %s", gvk.GroupVersion().String(), gvk.Kind, ans)
		}
	}
	sdk.Handle(consumerHandler)
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
