package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cenkalti/backoff/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	xerrors "github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/ttl256/gophermart-loyalty/internal/auth"
	"github.com/ttl256/gophermart-loyalty/internal/database"
	"github.com/ttl256/gophermart-loyalty/internal/domain"
	migrations "github.com/ttl256/gophermart-loyalty/internal/sql"
)

type DBStorage struct {
	db              *pgxpool.Pool
	queries         *database.Queries
	logger          *slog.Logger
	errorClassifier *PostgresErrorClassifier
}

func NewDBStorage(ctx context.Context, dsn string) (*DBStorage, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("creating pg pool: %w", err)
	}
	queries := database.New(pool)
	return &DBStorage{
		db:              pool,
		queries:         queries,
		logger:          slog.Default(),
		errorClassifier: NewPostgresErrorClassifier(),
	}, nil
}

func (m *DBStorage) Close() {
	m.db.Close()
}

func (m *DBStorage) CreateUser(
	ctx context.Context,
	user domain.User,
	passwordHash auth.PasswordHash,
) (uuid.UUID, error) {
	id, err := backoff.Retry(ctx, func() (uuid.UUID, error) {
		id, err := m.createUser(ctx, user, passwordHash)
		if err != nil {
			if m.errorClassifier.Classify(err) == Permanent {
				return uuid.UUID{}, backoff.Permanent(err)
			}
			return uuid.UUID{}, err
		}
		return id, nil
	})
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) {
			if pgError.Code == pgerrcode.UniqueViolation {
				return uuid.UUID{}, domain.ErrLoginExists
			}
		}
		return uuid.UUID{}, xerrors.WithStack(err)
	}
	return id, nil
}

func (m *DBStorage) createUser(
	ctx context.Context,
	user domain.User,
	passwordHash auth.PasswordHash,
) (uuid.UUID, error) {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = errors.Join(err, fmt.Errorf("rollback tx: %w", errRollback))
			}
		}
	}()
	qtx := m.queries.WithTx(tx)
	params := database.InsertUserParams{
		ID:           user.ID,
		Login:        user.Login,
		PasswordHash: string(passwordHash),
	}
	id, err := qtx.InsertUser(ctx, params)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("saving user: %w", err)
	}
	return id, nil
}

func (m *DBStorage) GetUserByLogin(ctx context.Context, login string) (domain.User, auth.PasswordHash, error) {
	user, err := backoff.Retry(ctx, func() (database.User, error) {
		user, err := m.getUserByLogin(ctx, login)
		if err != nil {
			if m.errorClassifier.Classify(err) == Permanent {
				return database.User{}, backoff.Permanent(err)
			}
			return database.User{}, err
		}
		return user, nil
	})
	if err != nil {
		m.logger.InfoContext(ctx, "getuser error", slog.Any("error", fmt.Sprintf("%+v", err)))
		if errors.Is(err, pgx.ErrNoRows) {
			m.logger.InfoContext(ctx, "getuser error no rows", slog.Any("error", fmt.Sprintf("%+v", err)))
			return domain.User{}, "", domain.ErrInvalidCredentials
		}
		return domain.User{}, "", fmt.Errorf("getting user: %w", err)
	}
	return domain.User{ID: user.ID, Login: user.Login}, auth.PasswordHash(user.PasswordHash), nil
}

func (m *DBStorage) getUserByLogin(ctx context.Context, login string) (database.User, error) {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return database.User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = errors.Join(err, fmt.Errorf("rollback tx: %w", errRollback))
			}
		}
	}()
	qtx := m.queries.WithTx(tx)
	dbUser, err := qtx.SelectUserByLogin(ctx, login)
	if err != nil {
		return database.User{}, fmt.Errorf("getting user: %w", err)
	}
	return dbUser, nil
}

func (m *DBStorage) RegisterOrder(ctx context.Context, userID uuid.UUID, order domain.OrderNumber) (uuid.UUID, error) {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = errors.Join(err, fmt.Errorf("rollback tx: %w", errRollback))
			}
		}
	}()
	qtx := m.queries.WithTx(tx)
	idInsert, err := qtx.InsertOrder(
		ctx, database.InsertOrderParams{
			Number:  string(order),
			UserID:  userID,
			Status:  domain.OrderStatusNEW.String(),
			Accrual: decimal.Decimal{},
		},
	)
	if err == nil {
		return idInsert, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, xerrors.WithStack(err)
	}
	id, err := qtx.GetOrderOwner(ctx, string(order))
	if err != nil {
		return uuid.UUID{}, xerrors.WithStack(err)
	}
	if id == userID {
		return userID, domain.ErrOrderAlreadyUploadedByUser
	}
	return id, domain.ErrOrderOwnedByAnotherUser
}

func (m *DBStorage) GetOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = errors.Join(err, fmt.Errorf("rollback tx: %w", errRollback))
			}
		}
	}()
	qtx := m.queries.WithTx(tx)
	dbOrders, err := qtx.GetOrders(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting orders: %w", err)
	}
	orders := make([]domain.Order, 0, len(dbOrders))
	for _, i := range dbOrders {
		var status domain.OrderStatus
		status, err = domain.ParseOrderStatus(i.Status)
		if err != nil {
			return nil, xerrors.WithStack(err)
		}
		order := domain.Order{
			Number:     domain.OrderNumber(i.Number),
			Status:     status,
			UserID:     userID,
			Accrual:    i.Accrual,
			UploadedAt: i.UploadedAt,
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (m *DBStorage) GetBalance(ctx context.Context, userID uuid.UUID) (domain.Balance, error) {
	row, err := m.queries.GetBalance(ctx, userID)
	if err != nil {
		return domain.Balance{}, fmt.Errorf("getting balance: %w", err)
	}
	return domain.Balance{Current: row.Current, Withdrawn: row.Withdrawn}, nil
}

func (m *DBStorage) RepoPing(ctx context.Context) error {
	var attempt int
	_, err := backoff.Retry(ctx, func() (bool, error) {
		attempt++
		m.logger.DebugContext(ctx, "connecting to db", slog.Int("attempt", attempt))
		err := m.db.Ping(ctx)
		if err != nil {
			m.logger.Error("failed connecting to db", slog.Int("attempt", attempt), slog.Any("error", err))
			return true, fmt.Errorf("ping db: %w", err)
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	m.logger.InfoContext(ctx, "connected to db", slog.Int("attempt", attempt))
	return nil
}

// MigrationAction ENUM(up, drop).
type MigrationAction int //nolint: recvcheck //fine

func (m *DBStorage) Migrate(action MigrationAction) error {
	iofsDriver, err := iofs.New(migrations.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("creating iofs driver: %w", err)
	}
	dbDriver, err := postgres.WithInstance(stdlib.OpenDBFromPool(m.db), new(postgres.Config))
	if err != nil {
		return fmt.Errorf("creating database driver with instance: %w", err)
	}
	defer dbDriver.Close()

	mig, err := migrate.NewWithInstance("iofs", iofsDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("instantiating migration: %w", err)
	}
	var op func() error
	switch action {
	case MigrationActionUp:
		op = mig.Up
	case MigrationActionDrop:
		op = mig.Drop
	}
	if err = op(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			m.logger.Error("performing database migration", slog.String("op", action.String()), slog.Any("error", err))
			return fmt.Errorf("performing %s database migration: %w", action.String(), err)
		}
		m.logger.Info("performing migration: no change")
		return nil
	}
	m.logger.Info("migration is complete", slog.String("op", action.String()))
	return nil
}
