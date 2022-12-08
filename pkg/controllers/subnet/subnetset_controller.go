package subnet

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"runtime"

	v1 "k8s.io/api/core/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	"github.com/vmware-tanzu/nsx-operator/pkg/metrics"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/subnet"
)

// SubnetSetReconciler SubnetSetReconciler reconciles a SubnetSet object
type SubnetSetReconciler struct {
	Client  client.Client
	Scheme  *apimachineryruntime.Scheme
	Service *subnet.SubnetService
}

func (r *SubnetSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &v1alpha1.SubnetSet{}
	log.Info("reconciling subnetset CR", "subnetset", req.NamespacedName)
	metrics.CounterInc(r.Service.NSXConfig, metrics.ControllerSyncTotal, MetricResTypeSubnetSet)

	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		log.Error(err, "unable to fetch security policy CR", "req", req.NamespacedName)
		return ResultNormal, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

func (r *SubnetReconciler) createSubnet(subnetset *v1alpha1.SubnetSet, name string) {
	if err := retry.OnError(retry.DefaultRetry, func(err error) bool {
		return err != nil
	}, func() error {
		blockDeletion := true
		obj := &v1alpha1.Subnet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: subnetset.Namespace,
				Name:      name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "v1alpha1",
						Kind:               "subnetsets.nsx.vmware.com",
						Name:               subnetset.Name,
						UID:                subnetset.UID,
						Controller:         nil,
						BlockOwnerDeletion: &blockDeletion,
					},
				},
			},
			Spec: v1alpha1.SubnetSpec{
				IPv4SubnetSize: subnetset.Spec.IPv4SubnetSize,
				AccessMode:     subnetset.Spec.AccessMode,
			},
		}
		if err := r.Client.Create(context.Background(), obj); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "failed to create subnet", "Namespace", subnetset.Namespace, "Name", name)
	}
}

func (r *SubnetSetReconciler) setupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SubnetSet{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: runtime.NumCPU(),
		}).
		Watches(&source.Kind{Type: &v1.Namespace{}}, &NamespaceHandler{Client: mgr.GetClient()}).
		Watches(&source.Kind{Type: &v1alpha1.VPC{}}, &VPCHandler{Client: mgr.GetClient()}).
		Complete(r)
}

// Start setup manager
func (r *SubnetSetReconciler) Start(mgr ctrl.Manager) error {
	err := r.setupWithManager(mgr)
	if err != nil {
		return err
	}
	return nil
}
