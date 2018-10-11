package fuse

import (
	"net"
	"strconv"
	"strings"

	errors3 "github.com/integr8ly/integration-controller/pkg/errors"

	v13 "github.com/openshift/api/route/v1"

	"k8s.io/api/core/v1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	errors2 "k8s.io/apimachinery/pkg/api/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"

	"github.com/pkg/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ServiceRouteConsumer struct {
	*FuseExistsChecker
	ns          string
	fuseCruder  Crudler
	routeClient routev1.RoutesGetter
}

const (
	serviceDiscoveryScheme = "discovery.syndesis/scheme"
	serviceDiscoveryPort   = "discovery.syndesis/port"
	serviceDiscoveryDoc    = "discovery.syndesis/description-path"
	serviceDiscoveryPath   = "discovery.syndesis/path"
)

func NewServiceRouteConsumer(ns string, routeClient routev1.RoutesGetter, fuseCruder Crudler) *ServiceRouteConsumer {
	return &ServiceRouteConsumer{
		FuseExistsChecker: NewFuseExistsChecker(ns, fuseCruder),
		ns:                ns,
		fuseCruder:        fuseCruder,
		routeClient:       routeClient,
	}
}

// Validate will check the runtime object being consumed is valid for creating the integration
func (src *ServiceRouteConsumer) Validate(object runtime.Object) error {
	return nil
}

func autoEnabled(meta v12.Object) bool {
	ann := meta.GetAnnotations()
	if _, ok := ann[serviceDiscoveryScheme]; !ok {
		return false
	}
	if _, ok := ann[serviceDiscoveryPort]; !ok {
		return false
	}
	return true
}

func schemeFromPort(port int32) string {
	https := []int32{8443, 443}
	http := []int32{80, 8080}
	for _, p := range https {
		if p == port {
			return "https"
		}
	}
	for _, p := range http {
		if p == port {
			return "http"
		}
	}
	return ""
}

func (src *ServiceRouteConsumer) connectionType(object runtime.Object, meta v12.Object) (string, error) {
	ann := meta.GetAnnotations()
	s := object.(*v1.Service)
	var connType string
	// look for annotation to tell us based on https://github.com/kubernetes/community/blob/master/contributors/design-proposals/network/service-discovery.md
	if v, ok := ann[serviceDiscoveryScheme]; ok {
		connType = v
	}
	if v, ok := ann[serviceDiscoveryDoc]; ok && v != "" {
		connType = "api"
	}
	if connType != "" {
		return connType, nil
	}
	// check for a route
	r, err := src.getRouteForService(*s)
	if err != nil && !errors3.IsNotFoundErr(err) {
		return "", err
	}

	if r != nil && (r.Spec.TLS == nil || r.Spec.TLS != nil && r.Spec.TLS.Termination == v13.TLSTerminationEdge) {
		return "http", nil
	} else if r != nil {
		return "https", nil
	}
	// still not found look for well known ports
	for _, p := range s.Spec.Ports {
		scheme := schemeFromPort(p.Port)
		if scheme != "" {
			return scheme, nil
		}
	}
	return "", errors.New("failed to determine connection type ")
}

func buildParamsForConnectionType(connType, host, path, descDoc string, port int32) map[string]string {
	//todo move out to const
	p := map[string]string{
		"scheme":               connType,
		"description-doc-path": descDoc,
		"api-path":             path,
		"host":                 host,
	}
	if port != 0 {
		p["port"] = strconv.Itoa(int(port))
	}
	return p
}

func isReachable(host string) (bool, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		if _, ok := err.(*net.DNSError); ok {
			return false, nil
		}
		return false, err
	}
	return len(ips) > 0, nil
}

func (src *ServiceRouteConsumer) getRouteForService(service v1.Service) (*v13.Route, error) {
	routes, err := src.routeClient.Routes(service.Namespace).List(v12.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, r := range routes.Items {
		if r.Spec.To.Kind == "Service" && r.Spec.To.Name == service.GetName() {
			return &r, nil
		}
	}
	return nil, &errors3.NotFoundErr{Resource: "Route"}
}

func (src *ServiceRouteConsumer) getHostPathAndPortForService(svc v1.Service) (string, string, int32, error) {
	host := svc.Name + "." + svc.Namespace + ".svc"
	path := "/"
	is, err := isReachable(host)
	if err != nil {
		logrus.Error("unable to complete dns lookup ", err)
	}
	var port int32
	if is {

		if len(svc.Spec.Ports) == 1 {
			port = svc.Spec.Ports[0].Port
		}
		return host, path, port, nil
	}
	r, err := src.getRouteForService(svc)
	if err != nil {
		return "", "", port, errors.Wrap(err, "failed to get route for service")
	}
	if r.Spec.Port != nil {
		port = r.Spec.Port.TargetPort.IntVal
	}
	return r.Spec.Host, r.Spec.Path, port, nil
}

func name(o runtime.Object) string {
	accessor := o.(v12.ObjectMetaAccessor)
	return accessor.GetObjectMeta().GetNamespace() + "-" + strings.ToLower(o.GetObjectKind().GroupVersionKind().Kind) + "-" + accessor.GetObjectMeta().GetName() + "-to-fuse"
}

// CreateAvailableIntegration sets up and creates a new integration object
func (src *ServiceRouteConsumer) CreateAvailableIntegration(object runtime.Object, targetNS string, enabled bool) error {
	logrus.Debug(" consumer: creating available integration")
	accessor := object.(v12.ObjectMetaAccessor)
	svc := object.(*v1.Service)

	host, path, port, err := src.getHostPathAndPortForService(*svc)
	if err != nil {
		return err
	}

	ingrtn := v1alpha1.NewIntegration(name(object))
	ingrtn.Namespace = targetNS
	ingrtn.Status.DiscoveryResource = v1alpha1.DiscoveryResource{Namespace: svc.Namespace, Name: svc.Name, GroupVersionKind: object.GetObjectKind().GroupVersionKind()}
	ingrtn.Spec.ServiceProvider = string(v1alpha1.FuseIntegrationTarget)
	ingrtn.Spec.Client = accessor.GetObjectMeta().GetNamespace() + "/" + accessor.GetObjectMeta().GetName()
	connType, err := src.connectionType(object, accessor.GetObjectMeta())
	if err != nil {
		logrus.Error("service_route consumer: failed to set up connection", err)
		return err
	}

	ann := accessor.GetObjectMeta().GetAnnotations()
	var swaggerDoc string
	if ann != nil {
		swaggerDoc = ann[serviceDiscoveryDoc]
	}
	if swaggerDoc == "" {
		swaggerDoc = ingrtn.Status.IntegrationMetaData["description-doc-path"]
	}
	ingrtn.Spec.IntegrationType = connType
	ingrtn.Spec.Enabled = autoEnabled(accessor.GetObjectMeta())
	ingrtn.Status.IntegrationMetaData = buildParamsForConnectionType(connType, host, path, swaggerDoc, port)
	if err := sdk.Create(ingrtn); err != nil && errors2.IsAlreadyExists(err) {
		cp := ingrtn.DeepCopy()
		err := sdk.Get(ingrtn, sdk.WithGetOptions(&v12.GetOptions{}))
		if err != nil {
			logrus.Error("service_route consumer: failed to set up integration", err)
			return err
		}
		logrus.Debug("updating integration to be ", ingrtn.Status.IntegrationMetaData)
		cp.ResourceVersion = ingrtn.ResourceVersion
		// copy over any user changes
		for k, v := range ingrtn.Status.IntegrationMetaData {
			if cp.Status.IntegrationMetaData[k] != v {
				cp.Status.IntegrationMetaData[k] = v
			}
		}
		cp.Spec.Enabled = ingrtn.Spec.Enabled
		return sdk.Update(cp)
	} else if err != nil && !errors2.IsAlreadyExists(err) {
		logrus.Error("service_route consumer: failed to set up integration", err)
	}

	logrus.Info("service_route consumer: setup integration ")
	return nil
}

// RemoveAvailableIntegration removed the integration object created by CreateAvailableIntegration
func (src *ServiceRouteConsumer) RemoveAvailableIntegration(object runtime.Object, targetNS string) error {
	intgration := v1alpha1.NewIntegration(name(object))
	intgration.Namespace = targetNS
	return src.fuseCruder.Delete(intgration)
}

// GVKs announces to the registry which objects this consumer is interested in
func (src *ServiceRouteConsumer) GVKs() []schema.GroupVersionKind {
	return []schema.GroupVersionKind{
		{
			Kind:    "Service",
			Group:   "",
			Version: "v1",
		},
	}
}
