package subnet

import (
	"context"
	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type NamespaceHandler struct {
	Client client.Client
}

func (h *NamespaceHandler) Create(e event.CreateEvent, _ workqueue.RateLimitingInterface) {
	log.V(4).Info("namespace create event, do nothing")
}

func (h *NamespaceHandler) Delete(e event.DeleteEvent, _ workqueue.RateLimitingInterface) {
	ns := e.Object.GetName()
	log.Info("cleaning Subnetset under Namespace", "Namespace", ns)
	if err := retry.OnError(retry.DefaultRetry, func(err error) bool {
		return err != nil
	}, func() error {
		if err := h.Client.DeleteAllOf(context.Background(), &v1alpha1.SubnetSet{}, client.InNamespace(ns)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "failed to delete SubnetSet", "Namespace", ns)
	}
}

func (h *NamespaceHandler) Generic(_ event.GenericEvent, _ workqueue.RateLimitingInterface) {
	log.V(4).Info("namespace generic event, do nothing")
}

func (h *NamespaceHandler) Update(_ event.UpdateEvent, _ workqueue.RateLimitingInterface) {
	log.V(4).Info("namespace update event, do nothing")
}
