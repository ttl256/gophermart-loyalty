package testutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type PG struct {
	Container *postgres.PostgresContainer
	DSN       string
}

func StartPG(ctx context.Context) (*PG, error) {
	c, err := postgres.Run(ctx,
		"postgres:18.1-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	if err != nil {
		return nil, fmt.Errorf("starting postgres container: %w", err)
	}

	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		errTerminate := c.Terminate(ctx)
		return nil, errors.Join(err, errTerminate)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		errTerminate := c.Terminate(ctx)
		return nil, errors.Join(err, errTerminate)
	}
	pool.Close()

	return &PG{Container: c, DSN: dsn}, nil
}

func (p *PG) Close(ctx context.Context) error {
	err := p.Container.Terminate(ctx)
	if err != nil {
		return fmt.Errorf("terminating postgres container: %w", err)
	}
	return nil
}
