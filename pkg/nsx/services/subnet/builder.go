package subnet

import (
	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
)

func (service *SubnetService) buildSubnet(obj *v1alpha1.Subnet) (*model.VpcSubnet, error) {
	nsxSubnet := &model.VpcSubnet{}
	return nsxSubnet, nil
}
