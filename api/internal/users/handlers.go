package users

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	repo "github.com/CoconutOrg/CoconutMaster/internal/adapters/sqlc"
	"github.com/CoconutOrg/CoconutMaster/internal/json"
	"github.com/CoconutOrg/CoconutMaster/internal/types"

	"github.com/go-chi/chi/v5"
)

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{
		service: service,
	}
}

func (h *handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.GetUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, users)
}

func (h *handler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID must be a number", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to fetch user: "+err.Error(), http.StatusBadRequest)
		return
	}

	json.Write(w, http.StatusOK, user)
}

func (h *handler) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUserByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "Failed to fetch user: "+err.Error(), http.StatusBadRequest)
		return
	}

	json.Write(w, http.StatusOK, user)
}

func (h *handler) GetUserByUsername(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "Failed to fetch user: "+err.Error(), http.StatusBadRequest)
		return
	}

	json.Write(w, http.StatusOK, user)
}

func (h *handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var registerUserParams RegisterUserParams
	if err := json.Read(r, &registerUserParams); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if (registerUserParams.Email == "") ||
		(registerUserParams.Password == "") ||
		(registerUserParams.Username == "") {
		err := fmt.Errorf("Missing parameter(s)")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.service.RegisterUser(r.Context(), registerUserParams)
	if err != nil && errors.Is(err, types.ErrAlreadyExists) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusCreated, user)
}

func (h *handler) RegisterConfirmUser(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, types.ErrNotFound.Error(), http.StatusBadRequest)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, types.ErrNotFound.Error(), http.StatusBadRequest)
		return
	}

	args := RegisterConfirmUserParams{
		Email: email,
		Code: code,
	}

	err := h.service.RegisterConfirmUser(r.Context(), args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	w.WriteHeader(http.StatusOK)
}

func (h *handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var loginUserParams LoginUserParams
	if err := json.Read(r, &loginUserParams); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if (loginUserParams.Email == "") || (loginUserParams.Password == "") {
		err := fmt.Errorf("Missing parameter(s)")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.service.LoginUser(r.Context(), loginUserParams)
	if err != nil && errors.Is(err, types.ErrNotFound) || (errors.Is(err, types.ErrInvalidCredentials)) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusCreated, result)
}

func (h *handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var createUserParams repo.CreateUserParams
	if err := json.Read(r, &createUserParams); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if (createUserParams.Email == "") ||
		(createUserParams.PasswordHash == "") ||
		(createUserParams.Username == "") {
		err := fmt.Errorf("Missing parameter(s)")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.service.CreateUser(r.Context(), createUserParams)
	if err != nil && errors.Is(err, types.ErrAlreadyExists) {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusCreated, user)
}

func (h *handler) UpdateUserById(w http.ResponseWriter, r *http.Request) {
	var updateUserByIdParams repo.UpdateUserByIdParams
	if err := json.Read(r, &updateUserByIdParams); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if (updateUserByIdParams.Email == "") ||
		(updateUserByIdParams.PasswordHash == "") ||
		(updateUserByIdParams.Username == "") {
		err := fmt.Errorf("Missing parameter(s)")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUserByID(r.Context(), updateUserByIdParams.ID)
	if err != nil || user == nil {
		http.Error(w, "Failed to delete user: "+err.Error(), http.StatusBadRequest)
		return
	}

	updatedUser, err := h.service.UpdateUserById(r.Context(), updateUserByIdParams)
	if err != nil {
		http.Error(w, "Failed to update user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, updatedUser)
}

func (h *handler) PatchUserRefreshTokenById(w http.ResponseWriter, r *http.Request) {
	var patchUserRefreshTokenByIdParams RefreshTokenParams
	if err := json.Read(r, &patchUserRefreshTokenByIdParams); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUserByID(r.Context(), patchUserRefreshTokenByIdParams.ID)
	if err != nil || user == nil {
		http.Error(w, "Failed to delete user: "+err.Error(), http.StatusBadRequest)
		return
	}

	refreshToken, err := h.service.PatchUserRefreshTokenById(r.Context(), patchUserRefreshTokenByIdParams)
	if err != nil {
		http.Error(w, "Failed to update user refresh token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, refreshToken)
}

func (h *handler) DeleteUserById(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID must be a number", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil || user == nil {
		http.Error(w, "Failed to delete user: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = h.service.DeleteUserById(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to delete user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
