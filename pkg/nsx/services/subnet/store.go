package subnet

import (
	"errors"

	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
)

// keyFunc is used to get the key of a resource, usually, which is the ID of the resource
func keyFunc(obj interface{}) (string, error) {
	switch v := obj.(type) {
	case model.VpcSubnet:
		return *v.Id, nil
	default:
		return "", errors.New("keyFunc doesn't support unknown type")
	}
}

// indexFunc is used to get index of a resource, usually, which is the UID of the CR controller reconciles,
// index is used to filter out resources which are related to the CR
func indexFunc(obj interface{}) ([]string, error) {
	filterTag := func(v []model.Tag) []string {
		res := make([]string, 0, 5)
		for _, tag := range v {
			if *tag.Scope == common.TagScopeSubnetCRUID {
				res = append(res, *tag.Tag)
			}
		}
		return res
	}
	res := make([]string, 0, 5)
	switch o := obj.(type) {
	case model.VpcSubnet:
		return filterTag(o.Tags), nil
	default:
		return res, errors.New("indexFunc doesn't support unknown type")
	}
}

// SubnetStore is a store for subnet.
type SubnetStore struct {
	common.ResourceStore
}

func (subnetStore *SubnetStore) Operate(i interface{}) error {
	if i == nil {
		return nil
	}
	subnet := i.(*model.VpcSubnet)
	if subnet.MarkedForDelete != nil && *subnet.MarkedForDelete {
		if err := subnetStore.Delete(*subnet); err != nil {
			return err
		}
		log.Info("Subnet deleted from store", "Subnet", subnet)
	} else {
		if err := subnetStore.Add(*subnet); err != nil {
			return err
		}
		log.Info("Subnet added to store", "Subnet", subnet)
	}
	return nil
}

func (subnetStore *SubnetStore) GetByIndex(key string, value string) []model.VpcSubnet {
	subnets := make([]model.VpcSubnet, 0)
	objs := subnetStore.ResourceStore.GetByIndex(key, value)
	for _, subnet := range objs {
		subnets = append(subnets, subnet.(model.VpcSubnet))
	}
	return subnets
}

func (subnetStore *SubnetStore) GetByKey(key string) *model.VpcSubnet {
	obj := subnetStore.ResourceStore.GetByKey(key)
	if obj == nil {
		return nil
	}
	subnet := obj.(model.VpcSubnet)
	return &subnet
}
