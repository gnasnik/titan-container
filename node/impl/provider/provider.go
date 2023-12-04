package provider

import (
	"context"

	"github.com/Filecoin-Titan/titan-container/api"
	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/google/uuid"
	"go.uber.org/fx"
)

var session = uuid.New()

// Provider represents a provider service in a cloud computing system.
type Provider struct {
	fx.In

	Client Client
}

var _ api.Provider = &Provider{}

func (p *Provider) Session(ctx context.Context) (uuid.UUID, error) {
	return session, nil
}

func (p *Provider) Version(context.Context) (api.Version, error) {
	return api.ProviderAPIVersion0, nil
}

func (p *Provider) GetStatistics(ctx context.Context) (*types.ResourcesStatistics, error) {
	return p.Client.GetStatistics(ctx)
}

func (p *Provider) GetDeployment(ctx context.Context, id types.DeploymentID) (*types.Deployment, error) {
	return p.Client.GetDeployment(ctx, id)
}

func (p *Provider) CreateDeployment(ctx context.Context, deployment *types.Deployment) error {
	return p.Client.CreateDeployment(ctx, deployment)
}

func (p *Provider) UpdateDeployment(ctx context.Context, deployment *types.Deployment) error {
	return p.Client.UpdateDeployment(ctx, deployment)
}

func (p *Provider) CloseDeployment(ctx context.Context, deployment *types.Deployment) error {
	return p.Client.CloseDeployment(ctx, deployment)
}

func (p *Provider) GetLogs(ctx context.Context, id types.DeploymentID) ([]*types.ServiceLog, error) {
	return p.Client.GetLogs(ctx, id)
}
func (p *Provider) GetEvents(ctx context.Context, id types.DeploymentID) ([]*types.ServiceEvent, error) {
	return p.Client.GetEvents(ctx, id)
}

func (p *Provider) GetDomains(ctx context.Context, id types.DeploymentID) ([]*types.DeploymentDomain, error) {
	return p.Client.GetDomains(ctx, id)
}

func (p *Provider) AddDomain(ctx context.Context, id types.DeploymentID, hostname string) error {
	return p.Client.AddDomain(ctx, id, hostname)
}

func (p *Provider) DeleteDomain(ctx context.Context, id types.DeploymentID, index int64) error {
	return p.Client.DeleteDomain(ctx, id, index)
}

func (p *Provider) ImportCertificate(ctx context.Context, id types.DeploymentID, cert *types.Certificate) error {
	return p.Client.ImportCertificate(ctx, id, cert)
}
