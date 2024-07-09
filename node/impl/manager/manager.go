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

var (
	ErrDeploymentNotFound = errors.New("deployment not found")
	ErrDomainAlreadyExist = errors.New("domain already exist")
	ErrInvalidAnnotations = errors.New("invalid annotations")
)

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

func (m *Manager) GetDeploymentList(ctx context.Context, opt *types.GetDeploymentOption) (*types.GetDeploymentListResp, error) {
	total, deployments, err := m.DB.GetDeployments(ctx, opt)
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

	return &types.GetDeploymentListResp{
		Deployments: deployments,
		Total:       total,
	}, nil
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

	if deploy.Owner != deployment.Owner {
		return errors.Errorf("update operation not allow")
	}

	deployment.ProviderID = deploy.ProviderID
	providerApi, err := m.ProviderManager.Get(deployment.ProviderID)
	if err != nil {
		return err
	}

	if len(deployment.Name) == 0 {
		deployment.Name = deploy.Name
	}

	if deployment.Expiration.IsZero() {
		deployment.Expiration = deploy.Expiration
	}

	deployment.CreatedAt = deploy.CreatedAt
	deployment.UpdatedAt = time.Now()

	for _, service := range deployment.Services {
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
		deploy, err := m.DB.GetDeploymentById(ctx, deployment.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("deployment not found")
		}

		if err != nil {
			return err
		}

		if deploy.Owner != deployment.Owner {
			return errors.Errorf("delete operation not allow")
		}

		providerApi, err := m.ProviderManager.Get(deploy.ProviderID)
		if err != nil {
			return err
		}

		err = providerApi.CloseDeployment(ctx, deploy)
		if err != nil {
			return err
		}

		return nil
	}

	if err := remoteClose(); err != nil && !force {
		return err
	}

	return m.DB.DeleteDeployment(ctx, deployment.ID)
}

func (m *Manager) GetLogs(ctx context.Context, deployment *types.Deployment) ([]*types.ServiceLog, error) {
	deploy, err := m.DB.GetDeploymentById(ctx, deployment.ID)
	if errors.As(err, sql.ErrNoRows) {
		return nil, ErrDeploymentNotFound
	}

	if err != nil {
		return nil, err
	}

	providerApi, err := m.ProviderManager.Get(deploy.ProviderID)
	if err != nil {
		return nil, err
	}

	serverLogs, err := providerApi.GetLogs(ctx, deployment.ID)
	if err != nil {
		return nil, err
	}

	for _, sl := range serverLogs {
		if len(sl.Logs) > 300 {
			sl.Logs = sl.Logs[len(sl.Logs)-300:]
		}
	}

	return serverLogs, nil
}

func (m *Manager) GetEvents(ctx context.Context, deployment *types.Deployment) ([]*types.ServiceEvent, error) {
	deploy, err := m.DB.GetDeploymentById(ctx, deployment.ID)
	if errors.As(err, sql.ErrNoRows) {
		return nil, ErrDeploymentNotFound
	}

	if err != nil {
		return nil, err
	}

	providerApi, err := m.ProviderManager.Get(deploy.ProviderID)
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
	if errors.As(err, sql.ErrNoRows) {
		return nil, ErrDeploymentNotFound
	}

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

	domains, err := providerApi.GetDomains(ctx, deploy.ID)
	if err != nil {
		return nil, err
	}

	for _, domain := range domains {
		if includeIP(domain.Name, provider.IP) {
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

func (m *Manager) AddDeploymentDomain(ctx context.Context, id types.DeploymentID, cert *types.Certificate) error {
	domain, err := m.DB.GetDomain(ctx, cert.Host)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if domain != nil && domain.DeploymentID == string(id) {
		return ErrDomainAlreadyExist
	}

	deploy, err := m.DB.GetDeploymentById(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrDeploymentNotFound
	}

	if err != nil {
		return err
	}

	providerApi, err := m.ProviderManager.Get(deploy.ProviderID)
	if err != nil {
		return err
	}

	err = providerApi.AddDomain(ctx, deploy.ID, cert)
	if err != nil {
		return err
	}

	return m.DB.AddDomain(ctx, &types.DeploymentDomain{
		DeploymentID: string(id),
		Name:         cert.Host,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
}

func (m *Manager) DeleteDeploymentDomain(ctx context.Context, id types.DeploymentID, domain string) error {
	deploy, err := m.DB.GetDeploymentById(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrDeploymentNotFound
	}

	if err != nil {
		return err
	}

	providerApi, err := m.ProviderManager.Get(deploy.ProviderID)
	if err != nil {
		return err
	}

	err = providerApi.DeleteDomain(ctx, deploy.ID, domain)
	if err != nil {
		return err
	}

	err = m.DB.DeleteDomain(ctx, domain)
	if err != nil {
		log.Errorf("delete domain: %v", err)
	}

	return nil
}

func (m *Manager) GetDeploymentShellEndpoint(ctx context.Context, id types.DeploymentID) (*types.ShellEndpoint, error) {
	deploy, err := m.DB.GetDeploymentById(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDeploymentNotFound
	}

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
		Scheme:    address.Scheme,
		Host:      address.Host,
		ShellPath: fmt.Sprintf("%s/%s", shellPath, deploy.ID),
	}

	return endpoint, nil
}

func (m *Manager) GetIngress(ctx context.Context, id types.DeploymentID) (*types.Ingress, error) {
	deploy, err := m.DB.GetDeploymentById(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDeploymentNotFound
	}

	if err != nil {
		return nil, err
	}

	providerApi, err := m.ProviderManager.Get(deploy.ProviderID)
	if err != nil {
		return nil, err
	}

	ingress, err := providerApi.GetIngress(ctx, deploy.ID)
	if err != nil {
		return nil, err
	}

	return ingress, nil
}

func (m *Manager) UpdateIngress(ctx context.Context, id types.DeploymentID, annotations map[string]string) error {
	if len(annotations) == 0 {
		return ErrInvalidAnnotations
	}

	deploy, err := m.DB.GetDeploymentById(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrDeploymentNotFound
	}

	if err != nil {
		return err
	}

	providerApi, err := m.ProviderManager.Get(deploy.ProviderID)
	if err != nil {
		return err
	}

	return providerApi.UpdateIngress(ctx, id, annotations)
}

var _ api.Manager = &Manager{}
