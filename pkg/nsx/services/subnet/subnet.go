package subnet

import (
	"errors"
	"os"
	"strings"
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
	changed := common.CompareResource(SubnetToComparable(existingSubnet), SubnetToComparable(nsxSubnet))
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
	log.Info("successfully deleted  nsxSubnet", "nsxSubnet", nsxSubnet)
	return nil
}

func (service *SubnetService) GetSubnetParamFromPath(nsxResourcePath string) *SubnetParameters {
	return &SubnetParameters{
		OrgID:     strings.Split(nsxResourcePath, "/")[len(nsxResourcePath)-5],
		ProjectID: strings.Split(nsxResourcePath, "/")[len(nsxResourcePath)-3],
		VPCID:     strings.Split(nsxResourcePath, "/")[len(nsxResourcePath)-1],
	}
}

func (service *SubnetService) GetSubnetStatus(subnet *v1alpha1.Subnet) (*model.VpcSubnetStatus, error) {
	nsxSubnet, err := service.buildSubnet(subnet)
	if err != nil {
		log.Error(err, "failed to build Subnet")
		return nil, err
	}
	param := service.GetSubnetParamFromPath(subnet.Status.NSXResourcePath)
	statusList, err := service.NSXClient.SubnetStatusClient.List(param.OrgID, param.ProjectID, param.VPCID, *nsxSubnet.Id)
	if err != nil {
		return nil, err
	}
	if len(statusList.Results) == 0 {
		err := errors.New("empty status result")
		log.Error(err, "no subnet status found")
		return nil, err
	}
	return &statusList.Results[0], nil
}

func (service *SubnetService) GetAvailableIPNum(subnet *v1alpha1.Subnet) (int64, error) {
	nsxSubnet, err := service.buildSubnet(subnet)
	if err != nil {
		log.Error(err, "failed to build Subnet")
		return -1, err
	}
	param := service.GetSubnetParamFromPath(subnet.Status.NSXResourcePath)
	ipPool, err := service.NSXClient.IPPoolClient.Get(param.OrgID, param.ProjectID, param.VPCID, *nsxSubnet.Id, ipPoolID)
	if err != nil {
		log.Error(err, "failed to get ip-pool")
		return -1, err
	}
	return *ipPool.PoolUsage.AvailableIps, nil
}
