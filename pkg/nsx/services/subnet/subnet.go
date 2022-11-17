package subnet

import (
	"github.com/vmware-tanzu/nsx-operator/pkg/logger"
	"github.com/vmware-tanzu/nsx-operator/pkg/nsx/services/common"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	"sync"
)

var (
	log                       = logger.Log
	MarkedForDelete           = true
	EnforceRevisionCheckParam = false
	ResourceTypeSubnet        = common.ResourceTypeSubnet
	NewConverter              = common.NewConverter
	// The following variables are defined as interface, they should be initialized as concrete type
	subnetStore common.Store
)

type SubnetService struct {
	common.Service
}

// InitializeSubnetService initialize Subnet service.
func InitializeSubnetService(service common.Service) (*SubnetService, error) {
	wg := sync.WaitGroup{}
	wgDone := make(chan bool)
	fatalErrors := make(chan error)
	subnetService := &SubnetService{Service: service}

	InitializeStore(subnetService)
	// TODO avoid use global variable
	subnetStore = &SubnetStore{
		ResourceStore: common.ResourceStore{
			Indexer:           subnetService.ResourceCacheMap[ResourceTypeSubnet],
			BindingType:       model.VpcSubnetBindingType(),
			ResourceAssertion: subnetAssertion,
		},
	}
	// TODO Maybe avoid use wg as there is only one goroutine.
	wg.Add(1)
	go subnetService.InitializeResourceStore(&wg, fatalErrors, ResourceTypeSubnet, subnetStore)
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
