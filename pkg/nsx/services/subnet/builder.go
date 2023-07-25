package subnet

import (
	"github.com/google/uuid"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
)

var (
	String = common.String
	Int64  = common.Int64
	Bool   = common.Bool
)

func getCluster(service *SubnetService) string {
	return service.NSXConfig.Cluster
}

func (service *SubnetService) buildSubnet(obj client.Object, tags []model.Tag) (*model.VpcSubnet, error) {
	tags = append(tags, service.buildBasicTags(obj)...)
	var nsxSubnet *model.VpcSubnet
	switch o := obj.(type) {
	case *v1alpha1.Subnet:
		nsxSubnet = &model.VpcSubnet{
			//TODO How to compare subnets when using random uuid.
			Id:             String(uuid.NewString()),
			AccessMode:     String(string(o.Spec.AccessMode)),
			DhcpConfig:     service.buildDHCPConfig(&o.Spec.DHCPConfig),
			Tags:           tags,
			AdvancedConfig: service.buildAdvancedConfig(&o.Spec.AdvancedConfig),
		}
	case *v1alpha1.SubnetSet:
		nsxSubnet = &model.VpcSubnet{
			//TODO How to compare subnets when using random uuid.
			Id:             String(uuid.NewString()),
			AccessMode:     String(string(o.Spec.AccessMode)),
			DhcpConfig:     service.buildDHCPConfig(&o.Spec.DHCPConfig),
			Tags:           tags,
			AdvancedConfig: service.buildAdvancedConfig(&o.Spec.AdvancedConfig),
		}
	default:
		return nil, SubnetTypeError
	}
	return nsxSubnet, nil
}

func (service *SubnetService) buildDHCPConfig(obj *v1alpha1.DHCPConfig) *model.DhcpConfig {
	// Subnet DHCP is used by AVI, not needed for now. We need to explicitly mark enableDhcp = false,
	// otherwise Subnet will use DhcpConfig inherited from VPC.
	dhcpConfig := &model.DhcpConfig{
		//DhcpRelayConfigPath: String(obj.DHCPRelayConfigPath),
		//DhcpV4PoolSize:      Int64(int64(obj.DHCPV4PoolSize)),
		//DhcpV6PoolSize:      Int64(int64(obj.DHCPV6PoolSize)),
		//DnsClientConfig:     service.buildDNSClientConfig(&obj.DNSClientConfig),
		EnableDhcp: Bool(false),
	}
	return dhcpConfig
}

func (service *SubnetService) buildDNSClientConfig(obj *v1alpha1.DNSClientConfig) *model.DnsClientConfig {
	dnsClientConfig := &model.DnsClientConfig{}
	dnsClientConfig.DnsServerIps = append(dnsClientConfig.DnsServerIps, obj.DNSServersIPs...)
	return dnsClientConfig
}

func (service *SubnetService) buildAdvancedConfig(obj *v1alpha1.AdvancedConfig) *model.SubnetAdvancedConfig {
	// Subnet uses static IP allocation, mark StaticIpAllocation = true.
	advancedConfig := &model.SubnetAdvancedConfig{
		StaticIpAllocation: &model.StaticIpAllocation{
			Enable: Bool(true),
		},
	}
	return advancedConfig
}

func (service *SubnetService) buildBasicTags(obj client.Object) []model.Tag {
	tags := []model.Tag{
		{
			Scope: String(common.TagScopeCluster),
			Tag:   String(getCluster(service)),
		},
		{
			Scope: String(common.TagScopeSubnetCRUID),
			Tag:   String(string(obj.GetUID())),
		},
		{
			Scope: String(common.TagScopeNamespace),
			Tag:   String(obj.GetNamespace()),
		},
	}
	switch obj.(type) {
	case *v1alpha1.Subnet:
		tags = append(tags, model.Tag{
			Scope: String(common.TagScopeSubnetCRType),
			Tag:   String("subnet"),
		})
	case *v1alpha1.SubnetSet:
		tags = append(tags, model.Tag{
			Scope: String(common.TagScopeSubnetCRType),
			Tag:   String("subnetset"),
		})
	default:
		log.Error(SubnetTypeError, "unsupported type when building NSX Subnet tags")
		return nil
	}
	return tags
}
