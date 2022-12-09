package subnet

import (
	"fmt"

	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
)

var (
	String = common.String
	Int64  = common.Int64
)

func (service *SubnetService) buildSubnet(obj *v1alpha1.Subnet) (*model.VpcSubnet, error) {
	nsxSubnet := &model.VpcSubnet{
		Id:          String(fmt.Sprintf("subnet_%s", obj.UID)),
		DisplayName: String(fmt.Sprintf("%s-%s", obj.ObjectMeta.Namespace, obj.ObjectMeta.Name)),
		AccessMode:  String(string(obj.Spec.AccessMode)),
	}
	nsxSubnet.IpAddresses = append(nsxSubnet.IpAddresses, obj.Spec.IPAddresses...)
	nsxSubnet.Tags = service.buildBasicTags(obj)
	return nsxSubnet, nil
}

func (service *SubnetService) buildBasicTags(obj *v1alpha1.Subnet) []model.Tag {
	tags := []model.Tag{
		{
			Scope: String(common.TagScopeCluster),
			Tag:   String(getCluster(service)),
		},
		{
			Scope: String(common.TagScopeNamespace),
			Tag:   String(obj.ObjectMeta.Namespace),
		},
		{
			Scope: String(common.TagScopeSubnetCRName),
			Tag:   String(obj.ObjectMeta.Name),
		},
		{
			Scope: String(common.TagScopeSubnetCRUID),
			Tag:   String(string(obj.UID)),
		},
	}
	return tags
}
