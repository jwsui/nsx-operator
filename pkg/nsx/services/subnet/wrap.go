package subnet

import (
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
)

// Patch API at infra level can be used in two flavours.
// 1. Like a regular API to update Infra object.
// 2. Hierarchical API: To create/update/delete entire or part of intent hierarchy Hierarchical.
// We use infra patch API in hierarchical mode to create/update/delete entire or part of intent hierarchy,
// for this convenience we can no longer CRUD CR separately, and reduce the number of API calls to NSX-T.

// WrapHierarchySubnet Wrap the subnet for InfraClient to patch.
func (service *SubnetService) WrapHierarchySubnet(subnet *model.VpcSubnet) (*model.Infra, error) {
	subnetChildren, err := service.wrapSubnet(subnet)
	if err != nil {
		return nil, err
	}
	var resourceReferenceChildren []*data.StructValue
	resourceReferenceChildren = append(resourceReferenceChildren, subnetChildren...)
	infraChildren, err := service.wrapResourceReference(resourceReferenceChildren)
	if err != nil {
		return nil, err
	}
	infra, err := service.wrapInfra(infraChildren)
	if err != nil {
		return nil, err
	}
	return infra, nil
}

func (service *SubnetService) wrapInfra(children []*data.StructValue) (*model.Infra, error) {
	// This is the outermost layer of the hierarchy subnet.
	// It doesn't need ID field.
	infraType := "Infra"
	infraObj := model.Infra{
		Children:     children,
		ResourceType: &infraType,
	}
	return &infraObj, nil
}

func (service *SubnetService) wrapSubnet(subnet *model.VpcSubnet) ([]*data.StructValue, error) {
	var subnetChildren []*data.StructValue
	subnet.ResourceType = &common.ResourceTypeSubnet // InfraClient need this field to identify the resource type
	childSubnet := model.ChildVpcSubnet{
		Id:              subnet.Id,
		MarkedForDelete: subnet.MarkedForDelete,
		ResourceType:    "ChildVpcSubnet",
		VpcSubnet:       subnet,
	}
	dataValue, errors := NewConverter().ConvertToVapi(childSubnet, model.ChildVpcSubnetBindingType())
	if len(errors) > 0 {
		return nil, errors[0]
	}
	subnetChildren = append(subnetChildren, dataValue.(*data.StructValue))
	return subnetChildren, nil
}

func (service *SubnetService) wrapResourceReference(children []*data.StructValue) ([]*data.StructValue, error) {
	var resourceReferenceChildren []*data.StructValue
	targetType := "Domain"
	id := getDomain(service)
	childDomain := model.ChildResourceReference{
		Id:           &id,
		ResourceType: "ChildResourceReference",
		TargetType:   &targetType,
		Children:     children,
	}
	dataValue, errors := NewConverter().ConvertToVapi(childDomain, model.ChildResourceReferenceBindingType())
	if len(errors) > 0 {
		return nil, errors[0]
	}
	resourceReferenceChildren = append(resourceReferenceChildren, dataValue.(*data.StructValue))
	return resourceReferenceChildren, nil
}

func getCluster(service *SubnetService) string {
	return service.NSXConfig.Cluster
}

func getDomain(service *SubnetService) string {
	return getCluster(service)
}
