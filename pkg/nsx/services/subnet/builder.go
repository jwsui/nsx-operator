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
	Bool   = common.Bool
)

func getCluster(service *SubnetService) string {
	return service.NSXConfig.Cluster
}

func (service *SubnetService) buildSubnet(obj *v1alpha1.Subnet) (*model.VpcSubnet, error) {
	nsxSubnet := &model.VpcSubnet{
		Id:             String(fmt.Sprintf("subnet_%s", obj.UID)),
		DisplayName:    String(fmt.Sprintf("%s-%s", obj.ObjectMeta.Namespace, obj.ObjectMeta.Name)),
		AccessMode:     String(string(obj.Spec.AccessMode)),
		DhcpConfig:     service.buildDHCPConfig(&obj.Spec.DHCPConfig),
		Tags:           service.buildBasicTags(obj),
		AdvancedConfig: service.buildAdvancedConfig(&obj.Spec.AdvancedConfig),
	}
	nsxSubnet.IpAddresses = append(nsxSubnet.IpAddresses, obj.Spec.IPAddresses...)
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
