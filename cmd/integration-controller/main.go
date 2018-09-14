package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/integr8ly/integration-controller/pkg/openshift"

	"github.com/integr8ly/integration-controller/pkg/fuse"

	"github.com/integr8ly/integration-controller/pkg/enmasse"

	"github.com/integr8ly/integration-controller/pkg/dispatch"
	"github.com/integr8ly/integration-controller/pkg/integration"
	_ "github.com/integr8ly/integration-controller/pkg/openshift"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
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

var (
	resyncPeriod int
	logLevel     string
)

func init() {
	flagset := flag.CommandLine
	flagset.IntVar(&resyncPeriod, "resync", 10, "change the resync period")
	flagset.StringVar(&logLevel, "log-level", logrus.Level.String(logrus.InfoLevel), "Log level to use. Possible values: panic, fatal, error, warn, info, debug")

}

func main() {
	flag.Parse()
	logLevel, err := logrus.ParseLevel(logLevel)
	logrus.Info("loglevel ", logLevel, resyncPeriod)
	if err != nil {
		logrus.Errorf("Failed to parse log level: %v", err)
	} else {
		logrus.SetLevel(logLevel)
	}
	printVersion()

	sdk.ExposeMetricsPort()

	resource := "integreatly.org/v1alpha1"
	kind := "Integration"
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("Failed to get watch namespace: %v", err)
	}

	resync, err := time.ParseDuration(fmt.Sprintf("%vs", resyncPeriod))
	if err != nil {
		panic(err)
	}

	k8Client := k8sclient.GetKubeClient()

	routeClient, err := routev1.NewForConfig(k8sclient.GetKubeConfig())
	if err != nil {
		panic(err)
	}

	// enmasseService
	enmasseService := enmasse.NewService(k8Client, routeClient.Routes(namespace), namespace)
	// fuse service
	fuseService := fuse.NewService()
	fuseIntegrationReconciler := integration.NewFuse(enmasseService, fuseService)
	integrationReconciler := integration.NewReconciler(fuseIntegrationReconciler)
	mainHandler := dispatch.NewHandler(k8Client)
	mainHandler.(*dispatch.Handler).AddHandler(integrationReconciler)
	mainHandler.(*dispatch.Handler).AddHandler(enmasse.NewReconciler())
	mainHandler.(*dispatch.Handler).AddHandler(openshift.NewRouteReconciler())
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)
	//watch user namespaces for routes
	userNamespaces := os.Getenv("WATCH_USER_NAMESPACES")
	if userNamespaces != "" {
		ns := strings.Split(userNamespaces, ",")
		for _, uNS := range ns {
			logrus.Info("watching user namespace ", uNS)
			sdk.Watch("route.openshift.io/v1", "Route", strings.TrimSpace(uNS), resync)
		}
	}
	sdk.Watch("v1", "ConfigMap", namespace, resync, sdk.WithLabelSelector("type=address-space"))
	sdk.Watch(resource, kind, namespace, resync)
	sdk.Handle(mainHandler)
	sdk.Run(context.TODO())
}
