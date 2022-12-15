package subnet

import (
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"

	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
)

type (
	Subnet     model.VpcSubnet
	Comparable common.Comparable
)

func (subnet *Subnet) Key() string {
	return *subnet.Id
}

func (subnet *Subnet) Value() data.DataValue {
	// IPv4SubnetSize/AccessMode/IPAddresses/DHCPConfig are immutable field,
	// it's not necessary to compare these fields.
	s := &Subnet{
		Id:             subnet.Id,
		DisplayName:    subnet.DisplayName,
		Tags:           subnet.Tags,
		AdvancedConfig: subnet.AdvancedConfig,
	}
	dataValue, _ := (*model.VpcSubnet)(s).GetDataValue__()
	return dataValue
}

func SubnetToComparable(subnet *model.VpcSubnet) Comparable {
	return (*Subnet)(subnet)
}
