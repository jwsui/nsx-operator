package subnetset

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
)

var defaultSubnetSets = []string{"DefaultVMSubnetSet", "DefaultPodSubnetSet"}

type VPCHandler struct {
	Client client.Client
}

func (h *VPCHandler) Create(e event.CreateEvent, _ workqueue.RateLimitingInterface) {
	ns := e.Object.GetNamespace()
	name := e.Object.GetName()
	log.Info("creating default Subnetset for VPC", "Namespace", ns, "Name", name)
	for _, subnetSet := range defaultSubnetSets {
		if err := retry.OnError(retry.DefaultRetry, func(err error) bool {
			return err != nil
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
			log.Error(err, "failed to create SubnetSet", "Namespace", ns, "Name", subnetSet)
		}
	}
}

func (h *VPCHandler) Delete(e event.DeleteEvent, _ workqueue.RateLimitingInterface) {
	ns := e.Object.GetName()
	log.Info("cleaning default Subnetset for VPC", "Namespace", e.Object.GetName())
	for _, subnetSet := range defaultSubnetSets {
		if err := retry.OnError(retry.DefaultRetry, func(err error) bool {
			return err != nil
		}, func() error {
			obj := &v1alpha1.SubnetSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      subnetSet,
				},
			}
			if err := h.Client.Delete(context.Background(), obj); err != nil {
				return client.IgnoreNotFound(err)
			}
			return nil
		}); err != nil {
			log.Error(err, "failed to create SubnetSet", "Namespace", ns, "Name", subnetSet)
		}
	}
}

func (h *VPCHandler) Generic(_ event.GenericEvent, _ workqueue.RateLimitingInterface) {
	log.V(4).Info("VPC generic event, do nothing")
}

func (h *VPCHandler) Update(_ event.UpdateEvent, _ workqueue.RateLimitingInterface) {
	log.V(4).Info("VPC update event, do nothing")
}
