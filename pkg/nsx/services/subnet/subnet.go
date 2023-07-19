package subnet

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

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
	// Default static ip-pool under Subnet.
	ipPoolID = "static-ipv4-default"
)

type SubnetService struct {
	common.Service
	SubnetStore *SubnetStore
}

// SubnetParameters stores parameters to CRUD Subnet object
type SubnetParameters struct {
	OrgID     string
	ProjectID string
	VPCID     string
}

var subnetService *SubnetService
var lock = &sync.Mutex{}

// GetSubnetService get singleton SubnetService instance, subnet/subnetset controller share the same instance.
func GetSubnetService(service common.Service) *SubnetService {
	if subnetService == nil {
		lock.Lock()
		defer lock.Unlock()
		if subnetService == nil {
			var err error
			if subnetService, err = InitializeSubnetService(service); err != nil {
				log.Error(err, "failed to initialize subnet commonService")
				os.Exit(1)
			}
		}
	}
	return subnetService
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

func (service *SubnetService) CreateOrUpdateSubnet(obj *v1alpha1.Subnet, orgID, projectID, vpcID string) error {
	nsxSubnet, err := service.buildSubnet(obj)
	if err != nil {
		log.Error(err, "failed to build Subnet")
		return err
	}
	existingSubnet := service.SubnetStore.GetByKey(*nsxSubnet.Id)
	changed := false
	if existingSubnet == nil {
		changed = true
	} else {
		changed = common.CompareResource(SubnetToComparable(existingSubnet), SubnetToComparable(nsxSubnet))
	}
	if !changed {
		log.Info("subnet not changed, skip updating", "subnet.Id", *nsxSubnet.Id)
		return nil
	}
	orgRoot, err := service.WrapHierarchySubnet(nsxSubnet, orgID, projectID, vpcID)
	if err != nil {
		log.Error(err, "WrapHierarchySubnet failed")
		return err
	}
	if err = service.NSXClient.OrgRootClient.Patch(*orgRoot, &EnforceRevisionCheckParam); err != nil {
		return err
	}
	if err = service.CheckRealizedState(nsxSubnet, common.RealizeTimeout); err != nil {
		return err
	}
	// Get Subnet from NSX after patch operation as NSX renders several fields like `path`/`parent_path`.
	if *nsxSubnet, err = service.NSXClient.SubnetsClient.Get(orgID, projectID, vpcID, *nsxSubnet.Id); err != nil {
		return err
	}
	if err = service.SubnetStore.Operate(nsxSubnet); err != nil {
		return err
	}
	log.Info("successfully updated nsxSubnet", "nsxSubnet", nsxSubnet)
	return nil
}

func (service *SubnetService) DeleteSubnet(obj interface{}, orgID, projectID, vpcID string) error {
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
	if err := service.DeleteIPAllocation(orgID, projectID, vpcID, *nsxSubnet.Id); err != nil {
		return err
	}
	for {
		log.Info("waiting for IP allocations to be released", "subnet", nsxSubnet.Id)
		usage, err := service.getIPPoolUsage(nsxSubnet)
		if err != nil {
			return err
		}
		if *usage.AllocatedIpAllocations > 0 {
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
	nsxSubnet.MarkedForDelete = &MarkedForDelete
	// WrapHighLevelSubnet will modify the input subnet, make a copy for the following store update.
	subnetCopy := *nsxSubnet
	orgRoot, err := service.WrapHierarchySubnet(nsxSubnet, orgID, projectID, vpcID)
	if err != nil {
		return err
	}
	if err = service.NSXClient.OrgRootClient.Patch(*orgRoot, &EnforceRevisionCheckParam); err != nil {
		return err
	}
	if err = service.SubnetStore.Operate(&subnetCopy); err != nil {
		return err
	}
	log.Info("successfully deleted nsxSubnet", "nsxSubnet", nsxSubnet)
	return nil
}

func (service *SubnetService) DeleteIPAllocation(orgID, projectID, vpcID, subnetID string) error {
	ipAllocations, err := service.NSXClient.IPAllocationClient.List(orgID, projectID, vpcID, subnetID, ipPoolID,
		nil, nil, nil, nil, nil, nil)
	if err != nil {
		log.Error(err, "failed to get ip-allocations", "Subnet", subnetID)
		return err
	}
	for _, alloc := range ipAllocations.Results {
		if err = service.NSXClient.IPAllocationClient.Delete(orgID, projectID, vpcID, subnetID, ipPoolID, *alloc.Id); err != nil {
			log.Error(err, "failed to delete ip-allocation", "Subnet", subnetID, "ip-alloc", *alloc.Id)
			return err
		}
	}
	log.Info("all IP allocations have been deleted", "Subnet", subnetID)
	return nil
}

func (service *SubnetService) GetSubnetParamFromPath(nsxResourcePath string) *SubnetParameters {
	pathArray := strings.Split(nsxResourcePath, "/")
	return &SubnetParameters{
		OrgID:     pathArray[2],
		ProjectID: pathArray[4],
		VPCID:     pathArray[6],
	}
}

func (service *SubnetService) GetSubnetStatus(subnet *model.VpcSubnet) (*model.VpcSubnetStatus, error) {
	param := service.GetSubnetParamFromPath(*subnet.Path)
	statusList, err := service.NSXClient.SubnetStatusClient.List(param.OrgID, param.ProjectID, param.VPCID, *subnet.Id)
	if err != nil {
		log.Error(err, "failed to get subnet status")
		return nil, err
	}
	if len(statusList.Results) == 0 {
		err := errors.New("empty status result")
		log.Error(err, "no subnet status found")
		return nil, err
	}
	return &statusList.Results[0], nil
}

func (service *SubnetService) getIPPoolUsage(nsxSubnet *model.VpcSubnet) (*model.PolicyPoolUsage, error) {
	param := service.GetSubnetParamFromPath(*nsxSubnet.Path)
	ipPool, err := service.NSXClient.IPPoolClient.Get(param.OrgID, param.ProjectID, param.VPCID, *nsxSubnet.Id, ipPoolID)
	if err != nil {
		log.Error(err, "failed to get ip-pool", "Subnet", *nsxSubnet.Id)
		return nil, err
	}
	return ipPool.PoolUsage, nil
}

func (service *SubnetService) GetIPPoolUsage(subnet *v1alpha1.Subnet) (*model.PolicyPoolUsage, error) {
	nsxSubnet, err := service.buildSubnet(subnet)
	if err != nil {
		log.Error(err, "failed to build Subnet", "Subnet", *nsxSubnet.Id)
		return nil, err
	}
	return service.getIPPoolUsage(nsxSubnet)
}

func (service *SubnetService) subnetRealized(nsxSubnet *model.VpcSubnet) bool {
	param := service.GetSubnetParamFromPath(*nsxSubnet.Path)
	results, err := service.NSXClient.RealizedStateClient.List(param.OrgID, param.ProjectID, *nsxSubnet.Path, nil)
	if err != nil {
		log.Error(err, "failed to check subnet realized status", "Subnet", *nsxSubnet.Id)
		return false
	}
	for _, result := range results.Results {
		if *result.EntityType != "RealizedLogicalSwitch" {
			continue
		}
		if *result.State == "REALIZED" {
			return true
		}
	}
	return false
}

func (service *SubnetService) CheckRealizedState(nsxSubnet *model.VpcSubnet, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ch := make(chan bool)
	go func(ctx context.Context, ch chan bool) {
		for {
			if service.subnetRealized(nsxSubnet) {
				ch <- true
				return
			}
			time.Sleep(time.Second)
		}
	}(ctx, ch)
	select {
	case <-ctx.Done():
		err := errors.New("realize timeout")
		log.Error(err, "timeout waiting for subnet realized", "subnet", *nsxSubnet.Id)
		return err
	case <-ch:
		return nil
	}
}
