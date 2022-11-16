package subnet

import (
	"context"
	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NamespaceHandler struct {
	Client client.Client
}

func (h *NamespaceHandler) Create(e event.CreateEvent, _ workqueue.RateLimitingInterface) {
	// TODO Check all log format.
	ns := e.Object.GetName()
	log.Log.Info("handling namespace create event", "Namespace", ns)
	subnetSets := []string{ns + "DefaultVMSubnetSet", ns + "DefaultPodSubnetSet"}
	for _, subnetSet := range subnetSets {
		if err := retry.OnError(retry.DefaultRetry, func(_ error) bool {
			return true
		}, func() error {
			obj := &v1alpha1.SubnetSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      subnetSet,
				},
				Spec: v1alpha1.SubnetSetSpec{},
			}
			if err := h.Client.Create(context.Background(), obj); err != nil {
				return err
			}
			return nil
		}); err != nil {
			log.Log.Error(err, "failed to create SubnetSet", "Namespace", ns, "Name", subnetSet)
		}
	}
}

func (h *NamespaceHandler) Delete(e event.DeleteEvent, _ workqueue.RateLimitingInterface) {
	ns := e.Object.GetName()
	log.Log.Info("handling namespace delete event", "Namespace", e.Object.GetName())
	if err := retry.OnError(retry.DefaultRetry, func(_ error) bool {
		return true
	}, func() error {
		if err := h.Client.DeleteAllOf(context.Background(), &v1alpha1.SubnetSet{}, client.InNamespace(ns)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Log.Error(err, "failed to delete SubnetSet", "Namespace", ns)
	}
}

func (h *NamespaceHandler) Generic(_ event.GenericEvent, _ workqueue.RateLimitingInterface) {
	log.Log.Info("namespace generic event, do nothing")
}

func (h *NamespaceHandler) Update(_ event.UpdateEvent, _ workqueue.RateLimitingInterface) {
	log.Log.Info("namespace update event, do nothing")
}
