package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	reposql "github.com/CoconutOrg/CoconutMaster/internal/adapters/sqlc"
	"github.com/CoconutOrg/CoconutMaster/internal/repo"
	auth "github.com/CoconutOrg/CoconutMaster/internal/services/auth"
	"github.com/CoconutOrg/CoconutMaster/internal/types"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service interface {
	GetUsers(ctx context.Context) ([]reposql.User, error)
	GetUserByID(ctx context.Context, id int64) (*reposql.User, error)
	GetUserByEmail(ctx context.Context, email string) (*reposql.User, error)
	GetUserByUsername(ctx context.Context, username string) (*reposql.User, error)
	RegisterUser(ctx context.Context, arg RegisterUserParams) (*reposql.User, error)
	RegisterConfirmUser(ctx context.Context, arg RegisterConfirmUserParams) (error)
	LoginUser(ctx context.Context, arg LoginUserParams) (*LoginUserResponse, error)
	LoginConfirmUser(ctx context.Context, arg CodeParams) (error)
	CreateUser(ctx context.Context, arg reposql.CreateUserParams) (*reposql.User, error)
	UpdateUserById(ctx context.Context, arg reposql.UpdateUserByIdParams) (*reposql.User, error)
	PatchUserRefreshTokenById(ctx context.Context, arg RefreshTokenParams) (*RefreshTokenResponse, error)
	PatchUserIsVerifiedById(ctx context.Context, id int64) (*reposql.User, error)
	DeleteUserById(ctx context.Context, id int64) error
}

type svc struct {
	reposql *reposql.Queries
	db   *pgx.Conn
	rdb  *repo.RdbRepository
	mr *repo.MqttRepository
}

type RegisterUserParams struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterConfirmUserParams struct {
	Email string `json:"email"`
	Code string `json:"code"`
}

type CodeParams struct {
	Code string `json:"code"`
}

type LoginUserParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginUserResponse struct {
	AuthToken string `json:"authToken"`
	RefreshToken string `json:"refreshToken"`
}

type RefreshTokenParams struct {
	ID int64 `json:"id"`
	RefreshToken string `json:"refreshToken"`
}

type RefreshTokenResponse struct {
	RefreshToken string `json:"refreshToken"`
}

func NewService(reposql *reposql.Queries, rdb *repo.RdbRepository, mr *repo.MqttRepository, db *pgx.Conn) Service {
	return &svc{
		reposql: reposql,
		db:   db,
		rdb: rdb,
		mr: mr,
	}
}

func (s *svc) GetUsers(ctx context.Context) ([]reposql.User, error) {
	return s.reposql.GetUsers(ctx)
}

func (s *svc) GetUserByID(ctx context.Context, id int64) (*reposql.User, error) {
	user, err := s.reposql.GetUserByID(ctx, id)
	return &user, err
}

func (s *svc) GetUserByEmail(ctx context.Context, email string) (*reposql.User, error) {
	user, err := s.reposql.GetUserByEmail(ctx, email)
	return &user, err
}

func (s *svc) GetUserByUsername(ctx context.Context, username string) (*reposql.User, error) {
	user, err := s.reposql.GetUserByUsername(ctx, username)
	return &user, err
}

func (s *svc) RegisterUser(ctx context.Context, arg RegisterUserParams) (*reposql.User, error) {
	hash, err := auth.HashPassword(arg.Password)
	if err != nil {
		return nil, err
	}

	createUserParams := reposql.CreateUserParams{
		Username:     arg.Username,
		Email:        arg.Email,
		PasswordHash: hash,
	}

	user, err := s.GetUserByEmail(ctx, arg.Email)
	if err != nil {
		// if user doesnt exist, create
		user, err = s.CreateUser(ctx, createUserParams)
		if err != nil {
			return user, err
		}
		slog.Info("User Created!")
	} else {
		// if user already exists, check credentials and send the code
		err = auth.ComparePasswords(arg.Password, user.PasswordHash)
		if err != nil {
			return nil, types.ErrInvalidCredentials
		}
	}
	
	idV4, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	slog.Info("uuid Created!")

	// delete user in case of failure to communicate with other services
	err = s.rdb.SetRegisterCodeByEmail(ctx, arg.Email, idV4.String(), 3*time.Minute)
	if err != nil {
		tx, err := s.db.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)

		user, err := s.GetUserByEmail(ctx, arg.Email)
		if err != nil {
			return nil, err
		}
		
		qtx := s.reposql.WithTx(tx)
		err = qtx.DeleteUserById(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		
		err = tx.Commit(ctx)
		if err != nil {
			return nil, err
		}

		return nil, err
	}
	slog.Info("Code Created!")

	err = s.mr.PublishRegiserConfirmUserMessage(arg.Email, idV4.String())
	if err != nil {
		tx, err := s.db.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)

		user, err := s.GetUserByEmail(ctx, arg.Email)
		if err != nil {
			return nil, err
		}
		
		qtx := s.reposql.WithTx(tx)
		err = qtx.DeleteUserById(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		
		err = tx.Commit(ctx)
		if err != nil {
			return nil, err
		}

		return nil, err
	}
	slog.Info("Code published!")

	return user, err
}

func (s *svc) RegisterConfirmUser(ctx context.Context, arg RegisterConfirmUserParams) (error) {
	cachedCode, _, err := s.rdb.GetRegisterCodeByEmail(ctx, arg.Email)
	if err != nil {
		return err
	}

	if cachedCode != arg.Code {
		return types.ErrInvalidCredentials
	}

	user, err := s.GetUserByEmail(ctx, arg.Email)
	if err != nil {
		return err
	}

	if user.IsVerified {
		return types.ErrAlreadyExists
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.reposql.WithTx(tx)
	_, err = qtx.PatchUserIsVerifiedById(ctx, reposql.PatchUserIsVerifiedByIdParams{ ID: user.ID, IsVerified: true })
	if err != nil {
		return err
	}
	
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *svc) LoginUser(ctx context.Context, arg LoginUserParams) (*LoginUserResponse, error) {
	user, err := s.GetUserByEmail(ctx, arg.Email)
	if err != nil {
		return nil, types.ErrNotFound
	}

	err = auth.ComparePasswords(arg.Password, user.PasswordHash)
	if err != nil {
		return nil, types.ErrInvalidCredentials
	}

	var token string
	token, err = auth.CreateJWT([]byte("secret"), user)
	if err != nil {
		return nil, err
	}
	
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if user.RefreshTokenExpiration.Time.Before(time.Now()) {
		refreshTokenBytes := make([]byte, 32)
		if _, err := rand.Read(refreshTokenBytes); err != nil {
			return nil, err
		}
		refreshToken := hex.EncodeToString(refreshTokenBytes)
		refreshTokenExpiration := pgtype.Timestamptz{
			Time: time.Now().AddDate(0, 1, 0),
			Valid: true,
			InfinityModifier: pgtype.Finite,
		}
		
		qtx := s.reposql.WithTx(tx)
		_, err = qtx.PatchUserRefreshTokenById(ctx, reposql.PatchUserRefreshTokenByIdParams{
			ID:                     user.ID,
			RefreshToken:           pgtype.Text{String: refreshToken, Valid: refreshToken != ""},
			RefreshTokenExpiration: refreshTokenExpiration,
		})
		if err != nil {
			return nil, err
		}
		
		err = tx.Commit(ctx)
		if err != nil {
			return nil, err
		}
	}

	return &LoginUserResponse{AuthToken: token, RefreshToken: user.RefreshToken.String}, err
}

func (s *svc) LoginConfirmUser(ctx context.Context, arg CodeParams) (error) {
	return nil
}

func (s *svc) CreateUser(ctx context.Context, arg reposql.CreateUserParams) (*reposql.User, error) {
	user, err := s.GetUserByUsername(ctx, arg.Username)
	if err == nil {
		return user, types.ErrAlreadyExists
	}

	user, err = s.GetUserByEmail(ctx, arg.Email)
	if err == nil {
		return user, types.ErrAlreadyExists
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.reposql.WithTx(tx)
	result, err := qtx.CreateUser(ctx, arg)
	if err != nil {
		return nil, err
	}
	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &result, err
}

func (s *svc) UpdateUserById(ctx context.Context, arg reposql.UpdateUserByIdParams) (*reposql.User, error) {
	if (arg.Email == "") || (arg.PasswordHash == "") || (arg.Username == "") {
		return nil, fmt.Errorf("Required parameter(s) missing!")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.reposql.WithTx(tx)
	result, err := qtx.UpdateUserById(ctx, arg)
	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &result, err
}

func (s *svc) PatchUserRefreshTokenById(ctx context.Context, arg RefreshTokenParams) (*RefreshTokenResponse, error) {
	if arg.RefreshToken == "" {
		return nil, fmt.Errorf("Required parameter(s) missing!")
	}

	user, err := s.GetUserByID(ctx, arg.ID)
	if err != nil {
		return nil, types.ErrNotFound
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if user.RefreshTokenExpiration.Time.Before(time.Now()) {
		refreshTokenBytes := make([]byte, 32)
		if _, err := rand.Read(refreshTokenBytes); err != nil {
			return nil, err
		}
		refreshToken := hex.EncodeToString(refreshTokenBytes)
		refreshTokenExpiration := pgtype.Timestamptz{
			Time: time.Now().AddDate(0, 1, 0),
			Valid: true,
			InfinityModifier: pgtype.Finite,
		}
		
		qtx := s.reposql.WithTx(tx)
		_, err = qtx.PatchUserRefreshTokenById(ctx, reposql.PatchUserRefreshTokenByIdParams{
			ID:                     user.ID,
			RefreshToken:           pgtype.Text{String: refreshToken, Valid: refreshToken != ""},
			RefreshTokenExpiration: refreshTokenExpiration,
		})
		if err != nil {
			return nil, err
		}
		
		err = tx.Commit(ctx)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	result := RefreshTokenResponse {
		RefreshToken: user.RefreshToken.String,
	}

	return &result, err
}

func (s *svc) PatchUserIsVerifiedById(ctx context.Context, id int64) (*reposql.User, error) {
	return nil, types.ErrNotFound
}

func (s *svc) DeleteUserById(ctx context.Context, id int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.reposql.WithTx(tx)
	err = qtx.DeleteUserById(ctx, id)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return err
}
