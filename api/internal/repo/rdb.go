package repo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRdbRepo(rdb *redis.Client) *RdbRepository {
	return &RdbRepository{
		db: rdb,
	}
}

func (rdb *RdbRepository) Close() error {
	return rdb.db.Close()
}

func NewRdbClient(addr string, port uint16) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", addr, port),

		DialerRetries: 5,
		DialerRetryTimeout: 100 * time.Millisecond,

		DialerRetryBackoff: redis.DialRetryBackoffExponential(100*time.Millisecond, 2*time.Second),
	})
	slog.Info("Connecting to redis db at", "addr", fmt.Sprintf("%s:%d", addr, port))
	return rdb
}

type IRdbRepository interface {
	SetRegisterCodeByEmail(ctx context.Context, email string, code string, lifetime time.Duration) (error)
	GetRegisterCodeByEmail(ctx context.Context, email string) (string, time.Duration, error)
	SetLoginCodeByEmail(ctx context.Context, email string, code string, lifetime time.Duration) (error)
	GetLoginCodeByEmail(ctx context.Context, email string) (string, time.Duration, error)
}

type RdbRepository struct {
	db *redis.Client
}

func (rdb *RdbRepository) SetRegisterCodeByEmail(ctx context.Context, email string, code string, lifetime time.Duration) (error) {
	err := rdb.db.Set(ctx, fmt.Sprintf("register_confirm_%s", email), code, lifetime).Err()
	return err
}

func (rdb *RdbRepository) GetRegisterCodeByEmail(ctx context.Context, email string) (string, time.Duration, error) {
	val, err := rdb.db.Get(ctx, fmt.Sprintf("register_confirm_%s", email)).Result()
	if err != nil {
		return val, 0, err
	}

	ttl, err := rdb.db.TTL(ctx, fmt.Sprintf("register_confirm_%s", email)).Result()

	return val, ttl, err
}

func (rdb *RdbRepository) SetLoginCodeByEmail(ctx context.Context, email string, code string, lifetime time.Duration) (error) {
	err := rdb.db.Set(ctx, fmt.Sprintf("login_confirm_%s", email), code, lifetime).Err()
	return err
}

func (rdb *RdbRepository) GetLoginCodeByEmail(ctx context.Context, email string) (string, time.Duration, error) {
	val, err := rdb.db.Get(ctx, fmt.Sprintf("login_confirm_%s", email)).Result()
	if err != nil {
		return val, 0, err
	}

	ttl, err := rdb.db.TTL(ctx, fmt.Sprintf("login_confirm_%s", email)).Result()

	return val, ttl, err
}