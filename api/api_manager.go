package api

import (
	"context"

	"github.com/Filecoin-Titan/titan-container/api/types"
)

// Manager is an interface for manager
type Manager interface {
	Common

	GetRemoteAddress(ctx context.Context) (string, error)                                                        //perm:read
	GetCertificate(context.Context) (*types.Certificate, error)                                                  //perm:read
	GetStatistics(ctx context.Context, id types.ProviderID) (*types.ResourcesStatistics, error)                  //perm:read
	ProviderConnect(ctx context.Context, url string, provider *types.Provider) error                             //perm:read
	GetProviderList(ctx context.Context, option *types.GetProviderOption) ([]*types.Provider, error)             //perm:read
	GetDeploymentList(ctx context.Context, opt *types.GetDeploymentOption) (*types.GetDeploymentListResp, error) //perm:read
	CreateDeployment(ctx context.Context, deployment *types.Deployment) error                                    //perm:admin
	UpdateDeployment(ctx context.Context, deployment *types.Deployment) error                                    //perm:admin
	CloseDeployment(ctx context.Context, deployment *types.Deployment, force bool) error                         //perm:admin
	GetLogs(ctx context.Context, deployment *types.Deployment) ([]*types.ServiceLog, error)                      //perm:read
	GetEvents(ctx context.Context, deployment *types.Deployment) ([]*types.ServiceEvent, error)                  //perm:read
	SetProperties(ctx context.Context, properties *types.Properties) error                                       //perm:admin
	GetDeploymentDomains(ctx context.Context, id types.DeploymentID) ([]*types.DeploymentDomain, error)          //perm:read
	AddDeploymentDomain(ctx context.Context, id types.DeploymentID, cert *types.Certificate) error               //perm:admin
	DeleteDeploymentDomain(ctx context.Context, id types.DeploymentID, domain string) error                      //perm:admin
	GetDeploymentShellEndpoint(ctx context.Context, id types.DeploymentID) (*types.ShellEndpoint, error)         //perm:admin
	GetIngress(ctx context.Context, id types.DeploymentID) (*types.Ingress, error)                               //perm:read
	UpdateIngress(ctx context.Context, id types.DeploymentID, annotations map[string]string) error               //perm:admin
}
