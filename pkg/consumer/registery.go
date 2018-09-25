package consumer

import (
	"sync"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/integr8ly/integration-controller/pkg/integration"
)

type consumers struct {
	consumerMap map[string][]integration.Consumer
	*sync.Mutex
}

var registered = &consumers{consumerMap: map[string][]integration.Consumer{}, Mutex: &sync.Mutex{}}

type Registry struct {
}

func (r Registry) RegisterConsumer(c integration.Consumer) {

	registered.Lock()
	defer registered.Unlock()

	for _, gvk := range c.GVKs() {
		key := gvk.String()
		if _, ok := registered.consumerMap[key]; ok {
			registered.consumerMap[key] = append(registered.consumerMap[key], c)
			continue
		}
		registered.consumerMap[key] = []integration.Consumer{c}
	}

	return
}

func (r Registry) ConsumersForKind(gvk schema.GroupVersionKind) []integration.Consumer {
	logrus.Debug("getting consumers for gvk ", gvk.String())
	var consumers []integration.Consumer
	key := gvk.String()
	consumers = registered.consumerMap[key]
	return consumers
}
