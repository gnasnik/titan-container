package db

import (
	"context"
	"github.com/Filecoin-Titan/titan-container/api/types"
)

func (m *ManagerDB) GetDomain(ctx context.Context, hostname string) (*types.DeploymentDomain, error) {
	qry := `SELECT * FROM domains where name = ?`
	var out types.DeploymentDomain
	if err := m.db.GetContext(ctx, &out, qry, hostname); err != nil {
		return nil, err
	}
	return &out, nil
}

func (m *ManagerDB) AddDomain(ctx context.Context, domain *types.DeploymentDomain) error {
	statement := `INSERT INTO domains (name, state, deployment_id, provider_id, created_at, updated_at) VALUES (:name, :state, :deployment_id, :provider_id, :created_at, :updated_at) 
		ON DUPLICATE KEY UPDATE deployment_id = VALUES(deployment_id), provider_id = values(provider_id), updated_at = NOW();`
	_, err := m.db.NamedExecContext(ctx, statement, domain)
	return err
}

func (m *ManagerDB) DeleteDomain(ctx context.Context, name string) error {
	statement := `DELETE from domains where name = ?`
	_, err := m.db.ExecContext(ctx, statement, name)
	return err
}
