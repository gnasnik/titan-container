// Code generated by titan/gen/api. DO NOT EDIT.

package api

import (
	"context"

	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/Filecoin-Titan/titan-container/journal/alerting"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

var ErrNotSupported = xerrors.New("method not supported")

type CommonStruct struct {
	Internal struct {
		AuthNew func(p0 context.Context, p1 []auth.Permission) ([]byte, error) `perm:"admin"`

		AuthVerify func(p0 context.Context, p1 string) ([]auth.Permission, error) `perm:"read"`

		Closing func(p0 context.Context) (<-chan struct{}, error) `perm:"admin"`

		Discover func(p0 context.Context) (types.OpenRPCDocument, error) `perm:"admin"`

		LogAlerts func(p0 context.Context) ([]alerting.Alert, error) `perm:"admin"`

		LogList func(p0 context.Context) ([]string, error) `perm:"admin"`

		LogSetLevel func(p0 context.Context, p1 string, p2 string) error `perm:"admin"`

		Session func(p0 context.Context) (uuid.UUID, error) `perm:"admin"`

		Shutdown func(p0 context.Context) error `perm:"admin"`

		Version func(p0 context.Context) (APIVersion, error) `perm:"read"`
	}
}

type CommonStub struct {
}

type ManagerStruct struct {
	CommonStruct

	Internal struct {
		AddDeploymentDomain func(p0 context.Context, p1 types.DeploymentID, p2 string) error `perm:"admin"`

		CloseDeployment func(p0 context.Context, p1 *types.Deployment, p2 bool) error `perm:"admin"`

		CreateDeployment func(p0 context.Context, p1 *types.Deployment) error `perm:"admin"`

		DeleteDeploymentDomain func(p0 context.Context, p1 types.DeploymentID, p2 string) error `perm:"admin"`

		GetDeploymentDomains func(p0 context.Context, p1 types.DeploymentID) ([]*types.DeploymentDomain, error) `perm:"read"`

		GetDeploymentList func(p0 context.Context, p1 *types.GetDeploymentOption) (*types.GetDeploymentListResp, error) `perm:"read"`

		GetDeploymentShellEndpoint func(p0 context.Context, p1 types.DeploymentID) (*types.ShellEndpoint, error) `perm:"admin"`

		GetEvents func(p0 context.Context, p1 *types.Deployment) ([]*types.ServiceEvent, error) `perm:"read"`

		GetLogs func(p0 context.Context, p1 *types.Deployment) ([]*types.ServiceLog, error) `perm:"read"`

		GetProviderList func(p0 context.Context, p1 *types.GetProviderOption) ([]*types.Provider, error) `perm:"read"`

		GetStatistics func(p0 context.Context, p1 types.ProviderID) (*types.ResourcesStatistics, error) `perm:"read"`

		ImportCertificate func(p0 context.Context, p1 types.DeploymentID, p2 *types.Certificate) error `perm:"admin"`

		ProviderConnect func(p0 context.Context, p1 string, p2 *types.Provider) error `perm:"admin"`

		SetProperties func(p0 context.Context, p1 *types.Properties) error `perm:"admin"`

		UpdateDeployment func(p0 context.Context, p1 *types.Deployment) error `perm:"admin"`
	}
}

type ManagerStub struct {
	CommonStub
}

type ProviderStruct struct {
	Internal struct {
		AddDomain func(p0 context.Context, p1 types.DeploymentID, p2 string) error `perm:"admin"`

		CloseDeployment func(p0 context.Context, p1 *types.Deployment) error `perm:"admin"`

		CreateDeployment func(p0 context.Context, p1 *types.Deployment) error `perm:"admin"`

		DeleteDomain func(p0 context.Context, p1 types.DeploymentID, p2 string) error `perm:"admin"`

		GetDeployment func(p0 context.Context, p1 types.DeploymentID) (*types.Deployment, error) `perm:"read"`

		GetDomains func(p0 context.Context, p1 types.DeploymentID) ([]*types.DeploymentDomain, error) `perm:"read"`

		GetEvents func(p0 context.Context, p1 types.DeploymentID) ([]*types.ServiceEvent, error) `perm:"read"`

		GetLogs func(p0 context.Context, p1 types.DeploymentID) ([]*types.ServiceLog, error) `perm:"read"`

		GetStatistics func(p0 context.Context) (*types.ResourcesStatistics, error) `perm:"read"`

		GetSufficientResourceNodes func(p0 context.Context, p1 *types.ComputeResources) ([]*types.SufficientResourceNode, error) `perm:"admin"`

		ImportCertificate func(p0 context.Context, p1 types.DeploymentID, p2 *types.Certificate) error `perm:"admin"`

		Session func(p0 context.Context) (uuid.UUID, error) `perm:"admin"`

		UpdateDeployment func(p0 context.Context, p1 *types.Deployment) error `perm:"admin"`

		Version func(p0 context.Context) (Version, error) `perm:"admin"`
	}
}

type ProviderStub struct {
}

func (s *CommonStruct) AuthNew(p0 context.Context, p1 []auth.Permission) ([]byte, error) {
	if s.Internal.AuthNew == nil {
		return *new([]byte), ErrNotSupported
	}
	return s.Internal.AuthNew(p0, p1)
}

func (s *CommonStub) AuthNew(p0 context.Context, p1 []auth.Permission) ([]byte, error) {
	return *new([]byte), ErrNotSupported
}

func (s *CommonStruct) AuthVerify(p0 context.Context, p1 string) ([]auth.Permission, error) {
	if s.Internal.AuthVerify == nil {
		return *new([]auth.Permission), ErrNotSupported
	}
	return s.Internal.AuthVerify(p0, p1)
}

func (s *CommonStub) AuthVerify(p0 context.Context, p1 string) ([]auth.Permission, error) {
	return *new([]auth.Permission), ErrNotSupported
}

func (s *CommonStruct) Closing(p0 context.Context) (<-chan struct{}, error) {
	if s.Internal.Closing == nil {
		return nil, ErrNotSupported
	}
	return s.Internal.Closing(p0)
}

func (s *CommonStub) Closing(p0 context.Context) (<-chan struct{}, error) {
	return nil, ErrNotSupported
}

func (s *CommonStruct) Discover(p0 context.Context) (types.OpenRPCDocument, error) {
	if s.Internal.Discover == nil {
		return *new(types.OpenRPCDocument), ErrNotSupported
	}
	return s.Internal.Discover(p0)
}

func (s *CommonStub) Discover(p0 context.Context) (types.OpenRPCDocument, error) {
	return *new(types.OpenRPCDocument), ErrNotSupported
}

func (s *CommonStruct) LogAlerts(p0 context.Context) ([]alerting.Alert, error) {
	if s.Internal.LogAlerts == nil {
		return *new([]alerting.Alert), ErrNotSupported
	}
	return s.Internal.LogAlerts(p0)
}

func (s *CommonStub) LogAlerts(p0 context.Context) ([]alerting.Alert, error) {
	return *new([]alerting.Alert), ErrNotSupported
}

func (s *CommonStruct) LogList(p0 context.Context) ([]string, error) {
	if s.Internal.LogList == nil {
		return *new([]string), ErrNotSupported
	}
	return s.Internal.LogList(p0)
}

func (s *CommonStub) LogList(p0 context.Context) ([]string, error) {
	return *new([]string), ErrNotSupported
}

func (s *CommonStruct) LogSetLevel(p0 context.Context, p1 string, p2 string) error {
	if s.Internal.LogSetLevel == nil {
		return ErrNotSupported
	}
	return s.Internal.LogSetLevel(p0, p1, p2)
}

func (s *CommonStub) LogSetLevel(p0 context.Context, p1 string, p2 string) error {
	return ErrNotSupported
}

func (s *CommonStruct) Session(p0 context.Context) (uuid.UUID, error) {
	if s.Internal.Session == nil {
		return *new(uuid.UUID), ErrNotSupported
	}
	return s.Internal.Session(p0)
}

func (s *CommonStub) Session(p0 context.Context) (uuid.UUID, error) {
	return *new(uuid.UUID), ErrNotSupported
}

func (s *CommonStruct) Shutdown(p0 context.Context) error {
	if s.Internal.Shutdown == nil {
		return ErrNotSupported
	}
	return s.Internal.Shutdown(p0)
}

func (s *CommonStub) Shutdown(p0 context.Context) error {
	return ErrNotSupported
}

func (s *CommonStruct) Version(p0 context.Context) (APIVersion, error) {
	if s.Internal.Version == nil {
		return *new(APIVersion), ErrNotSupported
	}
	return s.Internal.Version(p0)
}

func (s *CommonStub) Version(p0 context.Context) (APIVersion, error) {
	return *new(APIVersion), ErrNotSupported
}

func (s *ManagerStruct) AddDeploymentDomain(p0 context.Context, p1 types.DeploymentID, p2 string) error {
	if s.Internal.AddDeploymentDomain == nil {
		return ErrNotSupported
	}
	return s.Internal.AddDeploymentDomain(p0, p1, p2)
}

func (s *ManagerStub) AddDeploymentDomain(p0 context.Context, p1 types.DeploymentID, p2 string) error {
	return ErrNotSupported
}

func (s *ManagerStruct) CloseDeployment(p0 context.Context, p1 *types.Deployment, p2 bool) error {
	if s.Internal.CloseDeployment == nil {
		return ErrNotSupported
	}
	return s.Internal.CloseDeployment(p0, p1, p2)
}

func (s *ManagerStub) CloseDeployment(p0 context.Context, p1 *types.Deployment, p2 bool) error {
	return ErrNotSupported
}

func (s *ManagerStruct) CreateDeployment(p0 context.Context, p1 *types.Deployment) error {
	if s.Internal.CreateDeployment == nil {
		return ErrNotSupported
	}
	return s.Internal.CreateDeployment(p0, p1)
}

func (s *ManagerStub) CreateDeployment(p0 context.Context, p1 *types.Deployment) error {
	return ErrNotSupported
}

func (s *ManagerStruct) DeleteDeploymentDomain(p0 context.Context, p1 types.DeploymentID, p2 string) error {
	if s.Internal.DeleteDeploymentDomain == nil {
		return ErrNotSupported
	}
	return s.Internal.DeleteDeploymentDomain(p0, p1, p2)
}

func (s *ManagerStub) DeleteDeploymentDomain(p0 context.Context, p1 types.DeploymentID, p2 string) error {
	return ErrNotSupported
}

func (s *ManagerStruct) GetDeploymentDomains(p0 context.Context, p1 types.DeploymentID) ([]*types.DeploymentDomain, error) {
	if s.Internal.GetDeploymentDomains == nil {
		return *new([]*types.DeploymentDomain), ErrNotSupported
	}
	return s.Internal.GetDeploymentDomains(p0, p1)
}

func (s *ManagerStub) GetDeploymentDomains(p0 context.Context, p1 types.DeploymentID) ([]*types.DeploymentDomain, error) {
	return *new([]*types.DeploymentDomain), ErrNotSupported
}

func (s *ManagerStruct) GetDeploymentList(p0 context.Context, p1 *types.GetDeploymentOption) (*types.GetDeploymentListResp, error) {
	if s.Internal.GetDeploymentList == nil {
		return nil, ErrNotSupported
	}
	return s.Internal.GetDeploymentList(p0, p1)
}

func (s *ManagerStub) GetDeploymentList(p0 context.Context, p1 *types.GetDeploymentOption) (*types.GetDeploymentListResp, error) {
	return nil, ErrNotSupported
}

func (s *ManagerStruct) GetDeploymentShellEndpoint(p0 context.Context, p1 types.DeploymentID) (*types.ShellEndpoint, error) {
	if s.Internal.GetDeploymentShellEndpoint == nil {
		return nil, ErrNotSupported
	}
	return s.Internal.GetDeploymentShellEndpoint(p0, p1)
}

func (s *ManagerStub) GetDeploymentShellEndpoint(p0 context.Context, p1 types.DeploymentID) (*types.ShellEndpoint, error) {
	return nil, ErrNotSupported
}

func (s *ManagerStruct) GetEvents(p0 context.Context, p1 *types.Deployment) ([]*types.ServiceEvent, error) {
	if s.Internal.GetEvents == nil {
		return *new([]*types.ServiceEvent), ErrNotSupported
	}
	return s.Internal.GetEvents(p0, p1)
}

func (s *ManagerStub) GetEvents(p0 context.Context, p1 *types.Deployment) ([]*types.ServiceEvent, error) {
	return *new([]*types.ServiceEvent), ErrNotSupported
}

func (s *ManagerStruct) GetLogs(p0 context.Context, p1 *types.Deployment) ([]*types.ServiceLog, error) {
	if s.Internal.GetLogs == nil {
		return *new([]*types.ServiceLog), ErrNotSupported
	}
	return s.Internal.GetLogs(p0, p1)
}

func (s *ManagerStub) GetLogs(p0 context.Context, p1 *types.Deployment) ([]*types.ServiceLog, error) {
	return *new([]*types.ServiceLog), ErrNotSupported
}

func (s *ManagerStruct) GetProviderList(p0 context.Context, p1 *types.GetProviderOption) ([]*types.Provider, error) {
	if s.Internal.GetProviderList == nil {
		return *new([]*types.Provider), ErrNotSupported
	}
	return s.Internal.GetProviderList(p0, p1)
}

func (s *ManagerStub) GetProviderList(p0 context.Context, p1 *types.GetProviderOption) ([]*types.Provider, error) {
	return *new([]*types.Provider), ErrNotSupported
}

func (s *ManagerStruct) GetStatistics(p0 context.Context, p1 types.ProviderID) (*types.ResourcesStatistics, error) {
	if s.Internal.GetStatistics == nil {
		return nil, ErrNotSupported
	}
	return s.Internal.GetStatistics(p0, p1)
}

func (s *ManagerStub) GetStatistics(p0 context.Context, p1 types.ProviderID) (*types.ResourcesStatistics, error) {
	return nil, ErrNotSupported
}

func (s *ManagerStruct) ImportCertificate(p0 context.Context, p1 types.DeploymentID, p2 *types.Certificate) error {
	if s.Internal.ImportCertificate == nil {
		return ErrNotSupported
	}
	return s.Internal.ImportCertificate(p0, p1, p2)
}

func (s *ManagerStub) ImportCertificate(p0 context.Context, p1 types.DeploymentID, p2 *types.Certificate) error {
	return ErrNotSupported
}

func (s *ManagerStruct) ProviderConnect(p0 context.Context, p1 string, p2 *types.Provider) error {
	if s.Internal.ProviderConnect == nil {
		return ErrNotSupported
	}
	return s.Internal.ProviderConnect(p0, p1, p2)
}

func (s *ManagerStub) ProviderConnect(p0 context.Context, p1 string, p2 *types.Provider) error {
	return ErrNotSupported
}

func (s *ManagerStruct) SetProperties(p0 context.Context, p1 *types.Properties) error {
	if s.Internal.SetProperties == nil {
		return ErrNotSupported
	}
	return s.Internal.SetProperties(p0, p1)
}

func (s *ManagerStub) SetProperties(p0 context.Context, p1 *types.Properties) error {
	return ErrNotSupported
}

func (s *ManagerStruct) UpdateDeployment(p0 context.Context, p1 *types.Deployment) error {
	if s.Internal.UpdateDeployment == nil {
		return ErrNotSupported
	}
	return s.Internal.UpdateDeployment(p0, p1)
}

func (s *ManagerStub) UpdateDeployment(p0 context.Context, p1 *types.Deployment) error {
	return ErrNotSupported
}

func (s *ProviderStruct) AddDomain(p0 context.Context, p1 types.DeploymentID, p2 string) error {
	if s.Internal.AddDomain == nil {
		return ErrNotSupported
	}
	return s.Internal.AddDomain(p0, p1, p2)
}

func (s *ProviderStub) AddDomain(p0 context.Context, p1 types.DeploymentID, p2 string) error {
	return ErrNotSupported
}

func (s *ProviderStruct) CloseDeployment(p0 context.Context, p1 *types.Deployment) error {
	if s.Internal.CloseDeployment == nil {
		return ErrNotSupported
	}
	return s.Internal.CloseDeployment(p0, p1)
}

func (s *ProviderStub) CloseDeployment(p0 context.Context, p1 *types.Deployment) error {
	return ErrNotSupported
}

func (s *ProviderStruct) CreateDeployment(p0 context.Context, p1 *types.Deployment) error {
	if s.Internal.CreateDeployment == nil {
		return ErrNotSupported
	}
	return s.Internal.CreateDeployment(p0, p1)
}

func (s *ProviderStub) CreateDeployment(p0 context.Context, p1 *types.Deployment) error {
	return ErrNotSupported
}

func (s *ProviderStruct) DeleteDomain(p0 context.Context, p1 types.DeploymentID, p2 string) error {
	if s.Internal.DeleteDomain == nil {
		return ErrNotSupported
	}
	return s.Internal.DeleteDomain(p0, p1, p2)
}

func (s *ProviderStub) DeleteDomain(p0 context.Context, p1 types.DeploymentID, p2 string) error {
	return ErrNotSupported
}

func (s *ProviderStruct) GetDeployment(p0 context.Context, p1 types.DeploymentID) (*types.Deployment, error) {
	if s.Internal.GetDeployment == nil {
		return nil, ErrNotSupported
	}
	return s.Internal.GetDeployment(p0, p1)
}

func (s *ProviderStub) GetDeployment(p0 context.Context, p1 types.DeploymentID) (*types.Deployment, error) {
	return nil, ErrNotSupported
}

func (s *ProviderStruct) GetDomains(p0 context.Context, p1 types.DeploymentID) ([]*types.DeploymentDomain, error) {
	if s.Internal.GetDomains == nil {
		return *new([]*types.DeploymentDomain), ErrNotSupported
	}
	return s.Internal.GetDomains(p0, p1)
}

func (s *ProviderStub) GetDomains(p0 context.Context, p1 types.DeploymentID) ([]*types.DeploymentDomain, error) {
	return *new([]*types.DeploymentDomain), ErrNotSupported
}

func (s *ProviderStruct) GetEvents(p0 context.Context, p1 types.DeploymentID) ([]*types.ServiceEvent, error) {
	if s.Internal.GetEvents == nil {
		return *new([]*types.ServiceEvent), ErrNotSupported
	}
	return s.Internal.GetEvents(p0, p1)
}

func (s *ProviderStub) GetEvents(p0 context.Context, p1 types.DeploymentID) ([]*types.ServiceEvent, error) {
	return *new([]*types.ServiceEvent), ErrNotSupported
}

func (s *ProviderStruct) GetLogs(p0 context.Context, p1 types.DeploymentID) ([]*types.ServiceLog, error) {
	if s.Internal.GetLogs == nil {
		return *new([]*types.ServiceLog), ErrNotSupported
	}
	return s.Internal.GetLogs(p0, p1)
}

func (s *ProviderStub) GetLogs(p0 context.Context, p1 types.DeploymentID) ([]*types.ServiceLog, error) {
	return *new([]*types.ServiceLog), ErrNotSupported
}

func (s *ProviderStruct) GetStatistics(p0 context.Context) (*types.ResourcesStatistics, error) {
	if s.Internal.GetStatistics == nil {
		return nil, ErrNotSupported
	}
	return s.Internal.GetStatistics(p0)
}

func (s *ProviderStub) GetStatistics(p0 context.Context) (*types.ResourcesStatistics, error) {
	return nil, ErrNotSupported
}

func (s *ProviderStruct) GetSufficientResourceNodes(p0 context.Context, p1 *types.ComputeResources) ([]*types.SufficientResourceNode, error) {
	if s.Internal.GetSufficientResourceNodes == nil {
		return *new([]*types.SufficientResourceNode), ErrNotSupported
	}
	return s.Internal.GetSufficientResourceNodes(p0, p1)
}

func (s *ProviderStub) GetSufficientResourceNodes(p0 context.Context, p1 *types.ComputeResources) ([]*types.SufficientResourceNode, error) {
	return *new([]*types.SufficientResourceNode), ErrNotSupported
}

func (s *ProviderStruct) ImportCertificate(p0 context.Context, p1 types.DeploymentID, p2 *types.Certificate) error {
	if s.Internal.ImportCertificate == nil {
		return ErrNotSupported
	}
	return s.Internal.ImportCertificate(p0, p1, p2)
}

func (s *ProviderStub) ImportCertificate(p0 context.Context, p1 types.DeploymentID, p2 *types.Certificate) error {
	return ErrNotSupported
}

func (s *ProviderStruct) Session(p0 context.Context) (uuid.UUID, error) {
	if s.Internal.Session == nil {
		return *new(uuid.UUID), ErrNotSupported
	}
	return s.Internal.Session(p0)
}

func (s *ProviderStub) Session(p0 context.Context) (uuid.UUID, error) {
	return *new(uuid.UUID), ErrNotSupported
}

func (s *ProviderStruct) UpdateDeployment(p0 context.Context, p1 *types.Deployment) error {
	if s.Internal.UpdateDeployment == nil {
		return ErrNotSupported
	}
	return s.Internal.UpdateDeployment(p0, p1)
}

func (s *ProviderStub) UpdateDeployment(p0 context.Context, p1 *types.Deployment) error {
	return ErrNotSupported
}

func (s *ProviderStruct) Version(p0 context.Context) (Version, error) {
	if s.Internal.Version == nil {
		return *new(Version), ErrNotSupported
	}
	return s.Internal.Version(p0)
}

func (s *ProviderStub) Version(p0 context.Context) (Version, error) {
	return *new(Version), ErrNotSupported
}

var _ Common = new(CommonStruct)
var _ Manager = new(ManagerStruct)
var _ Provider = new(ProviderStruct)
