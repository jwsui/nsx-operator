package subnet

import (
	"github.com/vmware-tanzu/nsx-operator/pkg/apis/v1alpha1"
	"github.com/vmware-tanzu/nsx-operator/pkg/logger"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sync"
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
	subnetStore *SubnetStore
}

// InitializeSubnetService initialize Subnet service.
func InitializeSubnetService(service common.Service) (*SubnetService, error) {
	wg := sync.WaitGroup{}
	wgDone := make(chan bool)
	fatalErrors := make(chan error)
	subnetService := &SubnetService{
		Service: service,
		subnetStore: &SubnetStore{
			ResourceStore: common.ResourceStore{
				Indexer:     cache.NewIndexer(keyFunc, cache.Indexers{common.TagScopeSubnetCRUID: indexFunc}),
				BindingType: model.VpcSubnetBindingType(),
			},
		},
	}

	wg.Add(1)
	go subnetService.InitializeResourceStore(&wg, fatalErrors, ResourceTypeSubnet, subnetService.subnetStore)
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
		subnets := service.subnetStore.GetByIndex(common.TagScopeSubnetCRUID, string(subnet))
		if len(subnets) == 0 {
			log.Info("subnet is not found in store, skip deleting it", "uid", string(subnet))
			return nil
		}
		nsxSubnet = &subnets[0]
	}
	nsxSubnet.MarkedForDelete = &MarkedForDelete
	infraSubnet, err := service.WrapHierarchySubnet(nsxSubnet)
	if err != nil {
		return err
	}
	if err = service.NSXClient.InfraClient.Patch(*infraSubnet, &EnforceRevisionCheckParam); err != nil {
		return err
	}
	if err = service.subnetStore.Operate(nsxSubnet); err != nil {
		return err
	}
	log.Info("successfully deleted  nsxSubnet", "nsxSubnet", nsxSubnet)
	return nil
}
