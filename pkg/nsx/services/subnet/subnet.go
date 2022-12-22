package subnet

import (
	"sync"

	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	"github.com/vmware-tanzu/nsx-operator/pkg/logger"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
)

var (
	log                       = logger.Log
	MarkedForDelete           = true
	EnforceRevisionCheckParam = false
	ResourceTypeSubnet        = common.ResourceTypeSubnet
	NewConverter              = common.NewConverter
)

type SubnetService struct {
	common.Service
	SubnetStore *SubnetStore
}

// InitializeSubnetService initialize Subnet service.
func InitializeSubnetService(service common.Service) (*SubnetService, error) {
	wg := sync.WaitGroup{}
	wgDone := make(chan bool)
	fatalErrors := make(chan error)
	subnetService := &SubnetService{
		Service: service,
		SubnetStore: &SubnetStore{
			ResourceStore: common.ResourceStore{
				Indexer:     cache.NewIndexer(keyFunc, cache.Indexers{common.TagScopeSubnetCRUID: indexFunc}),
				BindingType: model.VpcSubnetBindingType(),
			},
		},
	}

	wg.Add(1)
	go subnetService.InitializeResourceStore(&wg, fatalErrors, ResourceTypeSubnet, subnetService.SubnetStore)
	go func() {
		wg.Wait()
		close(wgDone)
	}()
	select {
	case <-wgDone:
		break
	case err := <-fatalErrors:
		close(fatalErrors)
		return subnetService, err
	}
	return subnetService, nil
}

func (service *SubnetService) CreateOrUpdateSubnet(obj *v1alpha1.Subnet) error {
	nsxSubnet, err := service.buildSubnet(obj)
	if err != nil {
		log.Error(err, "failed to build Subnet")
		return err
	}
	existingSubnet := service.SubnetStore.GetByKey(*nsxSubnet.Id)
	changed := common.CompareResource(SubnetToComparable(existingSubnet), SubnetToComparable(nsxSubnet))
	if !changed {
		log.Info("subnet not changed, skip updating", "subnet.Id", *nsxSubnet.Id)
		return nil
	}
	//TODO replace org/project/vpc
	if err = service.NSXClient.SubnetClient.Patch("org", "project", "vpc", *nsxSubnet.Id, *nsxSubnet); err != nil {
		log.Error(err, "failed to patch subnet")
		return err
	}
	if err = service.SubnetStore.Operate(nsxSubnet); err != nil {
		return err
	}
	log.Info("successfully updated nsxSubnet", "nsxSubnet", nsxSubnet)
	return nil
}

func (service *SubnetService) DeleteSubnet(obj interface{}) error {
	var nsxSubnet *model.VpcSubnet
	switch subnet := obj.(type) {
	case *v1alpha1.Subnet:
		var err error
		nsxSubnet, err = service.buildSubnet(subnet)
		if err != nil {
			log.Error(err, "failed to build Subnet")
			return err
		}
	case types.UID:
		subnets := service.SubnetStore.GetByIndex(common.TagScopeSubnetCRUID, string(subnet))
		if len(subnets) == 0 {
			log.Info("subnet is not found in store, skip deleting it", "uid", string(subnet))
			return nil
		}
		nsxSubnet = &subnets[0]
	}
	nsxSubnet.MarkedForDelete = &MarkedForDelete
	if err := service.NSXClient.SubnetClient.Delete("org", "proj", "vpc", *nsxSubnet.Id); err != nil {
		log.Error(err, "failed to delete subnet")
	}
	if err := service.SubnetStore.Operate(nsxSubnet); err != nil {
		return err
	}
	log.Info("successfully deleted  nsxSubnet", "nsxSubnet", nsxSubnet)
	return nil
}

func (service *SubnetService) GetAvailableIPNum(subnet *v1alpha1.Subnet) (int64, error) {
	// TODO replace parameter
	// TODO support checking IPV6 IPAM
	if subnet.Spec.DHCPConfig.EnableDHCP {
		if dhcpStats, err := service.NSXClient.DHCPStatsClient.Get("org", "proj", "", "", nil, nil, nil, nil, nil, nil, nil); err != nil {
			log.Error(err, "error")
			return -1, err
		} else {
			return *dhcpStats.IpPoolStats[0].PoolSize - *dhcpStats.IpPoolStats[0].AllocatedNumber, nil
		}
	}
	if ipPool, err := service.NSXClient.IPPoolClient.Get("", "", "", "", ""); err != nil {
		log.Error(err, "error")
		return -1, err
	} else {
		return *ipPool.PoolUsage.AvailableIps, nil
	}
}
