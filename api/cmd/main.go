package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/CoconutOrg/CoconutMaster/internal/services/mqtt"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config{
		addr: "0.0.0.0",
		port: 4000,
		db: dbConfig{
			connectionString: "postgres://admin:admin4231@localhost:5432/coconut?sslmode=disable",
		},
	}

	db, err := pgx.Connect(ctx, cfg.db.connectionString)
	if err != nil {
		slog.Error("Error connecting to db", "error", err.Error())
		os.Exit(1)
	}
	defer db.Close(ctx)

	slog.Info("Connected to database", "connectionString", cfg.db.connectionString)

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6397",

		DialerRetries: 5,
		DialerRetryTimeout: 100 * time.Millisecond,

		DialerRetryBackoff: redis.DialRetryBackoffExponential(100*time.Millisecond, 2*time.Second),
	})
	defer rdb.Close()

	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://localhost:1883")
	opts.SetUsername("username")
	opts.SetClientID("id")
	opts.SetPassword("password")

	topics := make(map[string]byte)
	topics["user"] = byte(1)
	
	mqttClient, err := mqtt.NewMqttClient(opts, topics)
	if err != nil {
		slog.Error("Server has failed to connect to mqtt broker! Error", "error", err)
		os.Exit(1)
	}
	defer mqttClient.Disconnect()

	app := application{
		config: cfg,
		db: db,
		rdb: rdb,
		mqttc: mqttClient,
	}

	slog.SetDefault(logger)
	
	if err := app.run(app.mount()); err != nil {
		slog.Error("Server has failed to start! Error", "error", err)
		os.Exit(1)
	}
}