package subnet

import (
	"context"
	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

// SubnetSetReconciler SubnetSetReconciler reconciles a SubnetSet object
type SubnetReconciler struct {
	Client client.Client
	Scheme *apimachineryruntime.Scheme
}

func (r *SubnetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *SubnetReconciler) setupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Subnet{}).
		WithOptions(
			controller.Options{
				MaxConcurrentReconciles: runtime.NumCPU(),
			}).
		Complete(r)
}

// Start setup manager
func (r *SubnetReconciler) Start(mgr ctrl.Manager) error {
	err := r.setupWithManager(mgr)
	if err != nil {
		return err
	}
	return nil
}
