package api

import (
	"context"

	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/google/uuid"
)

type Provider interface {
	GetStatistics(ctx context.Context) (*types.ResourcesStatistics, error)                              //perm:read
	GetDeployment(ctx context.Context, id types.DeploymentID) (*types.Deployment, error)                //perm:read
	CreateDeployment(ctx context.Context, deployment *types.Deployment) error                           //perm:admin
	UpdateDeployment(ctx context.Context, deployment *types.Deployment) error                           //perm:admin
	CloseDeployment(ctx context.Context, deployment *types.Deployment) error                            //perm:admin
	GetLogs(ctx context.Context, id types.DeploymentID) ([]*types.ServiceLog, error)                    //perm:read
	GetEvents(ctx context.Context, id types.DeploymentID) ([]*types.ServiceEvent, error)                //perm:read
	GetDeploymentDomains(ctx context.Context, id types.DeploymentID) ([]*types.DeploymentDomain, error) //perm:read
	AddDeploymentDomain(ctx context.Context, id types.DeploymentID, hostname string) error              //perm:admin
	DeleteDeploymentDomain(ctx context.Context, id types.DeploymentID, index int64) error               //perm:admin

	Version(context.Context) (Version, error)   //perm:admin
	Session(context.Context) (uuid.UUID, error) //perm:admin
}
