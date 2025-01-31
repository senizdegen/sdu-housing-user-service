package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/senizdegen/sdu-housing/user-service/internal/config"
	"github.com/senizdegen/sdu-housing/user-service/internal/user"
	"github.com/senizdegen/sdu-housing/user-service/internal/user/db"
	"github.com/senizdegen/sdu-housing/user-service/pkg/cache/freecache"
	"github.com/senizdegen/sdu-housing/user-service/pkg/handlers/metric"
	"github.com/senizdegen/sdu-housing/user-service/pkg/logging"
	mongo "github.com/senizdegen/sdu-housing/user-service/pkg/mongodb"
	"github.com/senizdegen/sdu-housing/user-service/pkg/shutdown"
)

func main() {
	logging.Init()
	logger := logging.GetLogger()
	logger.Println("logger initialized")

	logger.Println("config initializing")
	cfg := config.GetConfig()
	logger.Println(cfg)

	logger.Println("router initializing")
	router := httprouter.New()

	metricHandler := metric.Handler{Logger: logger}
	metricHandler.Register(router)

	mongoClient, err := mongo.NewClient(
		context.Background(),
		cfg.MongoDB.Host,
		cfg.MongoDB.Port,
		cfg.MongoDB.Username,
		cfg.MongoDB.Password,
		cfg.MongoDB.Database,
		cfg.MongoDB.AuthDB,
	)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Println("cache initializing")
	refreshTokenCache := freecache.NewCacheRepo(104857600) // 100MB

	userStorage := db.NewStorage(mongoClient, cfg.MongoDB.Collection, logger)

	userService, err := user.NewService(userStorage, logger, refreshTokenCache)
	if err != nil {
		logger.Fatal(err)
	}

	usersHandler := user.Handler{
		Logger:      logger,
		UserService: userService,
	}

	usersHandler.Register(router)

	logger.Println("start application")
	start(router, logger, cfg)

}

func start(router http.Handler, logger logging.Logger, cfg *config.Config) {
	var server *http.Server
	var listener net.Listener

	if cfg.Listen.Type == "sock" {
		appDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			logger.Fatal(err)
		}
		socketPath := path.Join(appDir, "app.sock")
		logger.Infof("socket path: %s", socketPath)

		logger.Info("create and listen unix socket")
		listener, err = net.Listen("unix", socketPath)
		if err != nil {
			logger.Fatal(err)
		}
	} else {
		logger.Infof("bind application to host: %s and port: %s", cfg.Listen.BindIP, cfg.Listen.Port)

		var err error

		listener, err = net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.Listen.BindIP, cfg.Listen.Port))
		if err != nil {
			logger.Fatal(err)
		}
	}

	server = &http.Server{
		Handler:      router,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go shutdown.Graceful([]os.Signal{syscall.SIGABRT, syscall.SIGQUIT, syscall.SIGHUP, os.Interrupt, syscall.SIGTERM},
		server)

	logger.Println("application initialized and started")

	if err := server.Serve(listener); err != nil {
		switch {
		case errors.Is(err, http.ErrServerClosed):
			logger.Warn("server shutdown")
		default:
			logger.Fatal(err)
		}
	}
}
