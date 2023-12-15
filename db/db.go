package db

import (
	"context"
	_ "embed"
	logging "github.com/ipfs/go-log/v2"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"strings"
)

var log = logging.Logger("db")

func SqlDB(dsn string) (*sqlx.DB, error) {
	client, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err = client.Ping(); err != nil {
		return nil, err
	}

	// initialize the database
	log.Info("db: creating tables")
	err = createAllTables(context.Background(), client)
	if err != nil {
		return nil, errors.Errorf("failed to init db: %v", err)
	}

	return client, nil
}

//go:embed sql/db.sql
var createMainDBSQL string

func createAllTables(ctx context.Context, mainDB *sqlx.DB) error {
	lines := strings.Split(createMainDBSQL, ";")

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if _, err := mainDB.ExecContext(ctx, line); err != nil {
			return errors.Errorf("failed to create tables in main DB: %v", err)
		}
	}

	return nil
}
