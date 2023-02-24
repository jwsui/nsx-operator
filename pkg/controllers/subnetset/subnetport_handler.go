package subnetset

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
)

// SubnetPortHandler supports lazy-creation of Subnet, the first Subnet won't
// be created until there is a SubnetPort attached to it.
// - SubnetPort creation: get available Subnet for the SubnetPort, create new
//   Subnet if necessary.
// - SubnetPort deletion: if recycling Subnet is required, delete Subnets without
// SubnetPort attached to it.

type SubnetPortHandler struct {
	Reconciler *SubnetSetReconciler
}

// Create allocates Subnet for SubnetPort from SubnetSet.
func (h *SubnetPortHandler) Create(e event.CreateEvent, _ workqueue.RateLimitingInterface) {
	subnetPort := e.Object.(*v1alpha1.SubnetPort)
	if subnetPort.Spec.Subnet != "" {
		// Two possible scenarios:
		// - 1. User uses `.Spec.Subnet` directly instead of `.Spec.SubnetSet`.
		// - 2. Subnet has been allocated and `.Spec.Subnet` is rendered by SubnetPortHandler.
		return
	}
	subnetSet := &v1alpha1.SubnetSet{}
	key := types.NamespacedName{
		Namespace: subnetPort.GetNamespace(),
		Name:      subnetPort.Spec.SubnetSet,
	}
	if err := h.Reconciler.Client.Get(context.Background(), key, subnetSet); err != nil {
		log.Error(err, "failed to get SubnetSet", "ns", key.Namespace, "name", key.Name)
		return
	}
	log.Info("allocating Subnet for SubnetPort")
	allocatedSubnet, err := h.Reconciler.getAvailableSubnet(subnetSet)
	if err != nil {
		log.Error(err, "failed to allocate Subnet")
	}
	subnetPort.Spec.Subnet = allocatedSubnet.Name
	if err := h.Reconciler.Client.Update(context.Background(), subnetPort); err != nil {
		log.Error(err, "failed to update SubnetPort", "ns", subnetPort.Namespace, "name", subnetPort.Name)
	}
}

// Delete TODO Implement this method if required to recycle Subnet without SubnetPort attached.
func (h *SubnetPortHandler) Delete(e event.DeleteEvent, _ workqueue.RateLimitingInterface) {
	log.V(4).Info("SubnetPort generic event, do nothing")
}

func (h *SubnetPortHandler) Generic(_ event.GenericEvent, _ workqueue.RateLimitingInterface) {
	log.V(4).Info("SubnetPort generic event, do nothing")
}

func (h *SubnetPortHandler) Update(_ event.UpdateEvent, _ workqueue.RateLimitingInterface) {
	log.V(4).Info("SubnetPort update event, do nothing")
}

var SubnetPortPredicate = predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		return true
	},
	UpdateFunc: func(e event.UpdateEvent) bool {
		return false
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		// TODO When recycling Subnet is required, return true.
		return false
	},
	GenericFunc: func(genericEvent event.GenericEvent) bool {
		return false
	},
}