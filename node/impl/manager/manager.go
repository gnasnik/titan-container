package manager

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/Filecoin-Titan/titan-container/api"
	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/Filecoin-Titan/titan-container/db"
	"github.com/Filecoin-Titan/titan-container/node/handler"
	"github.com/Filecoin-Titan/titan-container/node/modules/dtypes"
	"github.com/google/uuid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"
	"go.uber.org/fx"
)

var log = logging.Logger("manager")

const shellPath = "/deployment/shell"

// Manager represents a manager service in a cloud computing system.
type Manager struct {
	fx.In

	api.Common
	DB *db.ManagerDB

	ProviderManager *ProviderManager

	SetManagerConfigFunc dtypes.SetManagerConfigFunc
	GetManagerConfigFunc dtypes.GetManagerConfigFunc
}

func (m *Manager) GetStatistics(ctx context.Context, id types.ProviderID) (*types.ResourcesStatistics, error) {
	providerApi, err := m.ProviderManager.Get(id)
	if err != nil {
		return nil, err
	}

	return providerApi.GetStatistics(ctx)
}

func (m *Manager) ProviderConnect(ctx context.Context, url string, provider *types.Provider) error {
	remoteAddr := handler.GetRemoteAddr(ctx)

	oldProvider, err := m.ProviderManager.Get(provider.ID)
	if err != nil && !errors.Is(err, ErrProviderNotExist) {
		return err
	}

	// close old provider
	if oldProvider != nil {
		m.ProviderManager.CloseProvider(provider.ID)
	}

	p, err := connectRemoteProvider(ctx, m, url)
	if err != nil {
		return errors.Errorf("connecting remote provider failed: %v", err)
	}

	log.Infof("Connected to a remote provider at %s, provider id %s", remoteAddr, provider.ID)

	err = m.ProviderManager.AddProvider(provider.ID, p, url)
	if err != nil {
		return err
	}

	if provider.IP == "" {
		provider.IP = strings.Split(remoteAddr, ":")[0]
	}

	provider.State = types.ProviderStateOnline
	provider.CreatedAt = time.Now()
	provider.UpdatedAt = time.Now()
	return m.DB.AddNewProvider(ctx, provider)
}

func (m *Manager) GetProviderList(ctx context.Context, opt *types.GetProviderOption) ([]*types.Provider, error) {
	return m.DB.GetAllProviders(ctx, opt)
}

func (m *Manager) GetDeploymentList(ctx context.Context, opt *types.GetDeploymentOption) ([]*types.Deployment, error) {
	deployments, err := m.DB.GetDeployments(ctx, opt)
	if err != nil {
		return nil, err
	}

	for _, deployment := range deployments {
		providerApi, err := m.ProviderManager.Get(deployment.ProviderID)
		if err != nil {
			deployment.State = types.DeploymentStateInActive
			continue
		}

		remoteDeployment, err := providerApi.GetDeployment(ctx, deployment.ID)
		if err != nil {
			continue
		}

		deployment.Services = remoteDeployment.Services
	}

	return deployments, nil
}

func (m *Manager) CreateDeployment(ctx context.Context, deployment *types.Deployment) error {
	providerApi, err := m.ProviderManager.Get(deployment.ProviderID)
	if err != nil {
		return err
	}

	// TODO: authority validation

	deployment.ID = types.DeploymentID(uuid.New().String())
	deployment.State = types.DeploymentStateActive
	deployment.CreatedAt = time.Now()
	deployment.UpdatedAt = time.Now()
	if deployment.Expiration.IsZero() {
		deployment.Expiration = time.Now().AddDate(0, 1, 0)
	}

	err = providerApi.CreateDeployment(ctx, deployment)
	if err != nil {
		return err
	}

	successDeployment, err := providerApi.GetDeployment(ctx, deployment.ID)
	if err != nil {
		return err
	}

	deployment.Services = successDeployment.Services
	for _, service := range deployment.Services {
		service.DeploymentID = deployment.ID
		service.CreatedAt = time.Now()
		service.UpdatedAt = time.Now()
	}

	err = m.DB.CreateDeployment(ctx, deployment)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) UpdateDeployment(ctx context.Context, deployment *types.Deployment) error {
	deploy, err := m.DB.GetDeploymentById(ctx, deployment.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("deployment not found")
	}

	if err != nil {
		return err
	}

	deployment.ProviderID = deploy.ProviderID
	providerApi, err := m.ProviderManager.Get(deployment.ProviderID)
	if err != nil {
		return err
	}

	for _, service := range deployment.Services {
		if service.Name == "" {
			service.Name = deployment.Name
		}
		service.DeploymentID = deployment.ID
		service.CreatedAt = time.Now()
		service.UpdatedAt = time.Now()
	}

	err = providerApi.UpdateDeployment(ctx, deployment)
	if err != nil {
		return err
	}

	err = m.DB.CreateDeployment(ctx, deployment)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) CloseDeployment(ctx context.Context, deployment *types.Deployment, force bool) error {
	remoteClose := func() error {
		providerApi, err := m.ProviderManager.Get(deployment.ProviderID)
		if err != nil {
			return err
		}

		err = providerApi.CloseDeployment(ctx, deployment)
		if err != nil {
			return err
		}

		return nil
	}

	if err := remoteClose(); err != nil && !force {
		return err
	}

	return m.DB.UpdateDeploymentState(ctx, deployment.ID, types.DeploymentStateClose)
}

func (m *Manager) GetLogs(ctx context.Context, deployment *types.Deployment) ([]*types.ServiceLog, error) {
	providerApi, err := m.ProviderManager.Get(deployment.ProviderID)
	if err != nil {
		return nil, err
	}

	return providerApi.GetLogs(ctx, deployment.ID)
}

func (m *Manager) GetEvents(ctx context.Context, deployment *types.Deployment) ([]*types.ServiceEvent, error) {
	providerApi, err := m.ProviderManager.Get(deployment.ProviderID)
	if err != nil {
		return nil, err
	}

	return providerApi.GetEvents(ctx, deployment.ID)
}

func (m *Manager) SetProperties(ctx context.Context, properties *types.Properties) error {
	_, err := m.ProviderManager.Get(properties.ProviderID)
	if err != nil {
		return err
	}

	properties.CreatedAt = time.Now()
	properties.UpdatedAt = time.Now()
	return m.DB.AddProperties(ctx, properties)
}

const (
	StateInvalid = "Invalid"
	StateOk      = "OK"
)

func (m *Manager) GetDeploymentDomains(ctx context.Context, id types.DeploymentID) ([]*types.DeploymentDomain, error) {
	deploy, err := m.DB.GetDeploymentById(ctx, id)
	if err != nil {
		return nil, err
	}

	providerApi, err := m.ProviderManager.Get(deploy.ProviderID)
	if err != nil {
		return nil, err
	}

	provider, err := m.DB.GetProviderById(ctx, deploy.ProviderID)
	if err != nil {
		return nil, err
	}

	domains, err := providerApi.GetDeploymentDomains(ctx, deploy.ID)
	if err != nil {
		return nil, err
	}

	for _, domain := range domains {
		if includeIP(domain.Host, provider.HostURI) {
			domain.State = StateOk
		} else {
			domain.State = StateInvalid
		}
	}

	return domains, nil
}

func includeIP(hostname string, expectedIP string) bool {
	ips, err := net.LookupHost(hostname)
	if err != nil {
		return false
	}

	for _, ip := range ips {
		if ip == expectedIP {
			return true
		}
	}

	return false
}

func (m *Manager) AddDeploymentDomain(ctx context.Context, id types.DeploymentID, hostname string) error {
	deploy, err := m.DB.GetDeploymentById(ctx, id)
	if err != nil {
		return err
	}

	providerApi, err := m.ProviderManager.Get(deploy.ProviderID)
	if err != nil {
		return err
	}

	return providerApi.AddDeploymentDomain(ctx, deploy.ID, hostname)
}

func (m *Manager) DeleteDeploymentDomain(ctx context.Context, id types.DeploymentID, index int64) error {
	deploy, err := m.DB.GetDeploymentById(ctx, id)
	if err != nil {
		return err
	}

	providerApi, err := m.ProviderManager.Get(deploy.ProviderID)
	if err != nil {
		return err
	}

	return providerApi.DeleteDeploymentDomain(ctx, deploy.ID, index)
}

func (m *Manager) GetDeploymentShellEndpoint(ctx context.Context, id types.DeploymentID) (*types.ShellEndpoint, error) {
	deploy, err := m.DB.GetDeploymentById(ctx, id)
	if err != nil {
		return nil, err
	}

	remoteAddr, err := m.ProviderManager.GetRemoteAddr(deploy.ProviderID)
	if err != nil {
		return nil, err
	}

	address, err := url.Parse(remoteAddr)
	if err != nil {
		return nil, err
	}

	endpoint := &types.ShellEndpoint{
		Host:      address.Host,
		ShellPath: fmt.Sprintf("%s/%s", shellPath, deploy.ID),
	}

	return endpoint, nil
}

var _ api.Manager = &Manager{}
