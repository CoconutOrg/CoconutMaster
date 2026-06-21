package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	reposql "github.com/CoconutOrg/CoconutMaster/internal/adapters/sqlc"
	"github.com/CoconutOrg/CoconutMaster/internal/middlewares"
	"github.com/CoconutOrg/CoconutMaster/internal/repo"
	"github.com/CoconutOrg/CoconutMaster/internal/services/mqtt"
	"github.com/CoconutOrg/CoconutMaster/internal/users"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	
	r.Use(middleware.Timeout(60 * time.Second))
	
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Healthy!"))
	})

	repository := reposql.New(app.db)
	rdbRepo := repo.NewRdbRepo(app.rdb)
	mqttRepo := repo.NewMqttRepo(app.mqttc)

	jwtSecret := []byte("secret")

	userService := users.NewService(repository, rdbRepo, mqttRepo, app.db)
	userHandler := users.NewHandler(userService)
	r.Route("/users", func(r chi.Router) {
		r.Post("/register", userHandler.RegisterUser)
		r.Get("/register/confirm", userHandler.RegisterConfirmUser)
		r.Put("/login", userHandler.LoginUser)

		r.Group(func(r chi.Router) {
			r.Use(middlewares.JwtAuthentication(jwtSecret, repository))
			r.Get("/", userHandler.GetUsers)
			r.Get("/{id}", userHandler.GetUserByID)
			r.Get("/email/{email}", userHandler.GetUserByEmail)
			r.Get("/username/{username}", userHandler.GetUserByUsername)
			r.Post("/", userHandler.CreateUser)
			r.Put("/", userHandler.UpdateUserById)
			r.Patch("/", userHandler.PatchUserRefreshTokenById)
			r.Delete("/{id}", userHandler.DeleteUserById)
		})
	})

	return r
}

func (app *application) run(handler http.Handler) error {
	srv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", app.config.addr, app.config.port),
		Handler: handler,
		WriteTimeout: time.Second * 30,
		ReadTimeout: time.Second * 10,
		IdleTimeout: time.Minute,
	}

	slog.Info(fmt.Sprintf("Server has started at addr %s:%d\n", app.config.addr, app.config.port))

	return srv.ListenAndServe()
}

type application struct {
	config config
	db     *pgx.Conn
	rdb *redis.Client
	mqttc *mqtt.MqttClient
}

type config struct {
	addr string
	port int
	db dbConfig
}

type dbConfig struct {
	connectionString string
}