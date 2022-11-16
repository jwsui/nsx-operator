package subnet

import (
	"context"
	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// SubnetSetReconciler SubnetSetReconciler reconciles a SubnetSet object
type SubnetSetReconciler struct {
	Client client.Client
	Scheme *apimachineryruntime.Scheme
}

func (r *SubnetSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *SubnetSetReconciler) setupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SubnetSet{}).
		WithOptions(
			controller.Options{
				MaxConcurrentReconciles: runtime.NumCPU(),
			}).
		Watches(&source.Kind{Type: &v1.Namespace{}},
			&handler.EnqueueRequestForObject{},
		).Complete(r)
	// TODO watch VPC CRD to creating default 'DefaultVMSubnetSet' and 'DefaultPodSubnetSet'.
}

// Start setup manager
func (r *SubnetSetReconciler) Start(mgr ctrl.Manager) error {
	err := r.setupWithManager(mgr)
	if err != nil {
		return err
	}
	return nil
}
