package main

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pikachu0310/livekit-server/internal/handler"
	"github.com/pikachu0310/livekit-server/internal/migration"
	"github.com/pikachu0310/livekit-server/internal/pkg/config"
	"github.com/pikachu0310/livekit-server/internal/repository"
	"github.com/pikachu0310/livekit-server/openapi"
	"net/http"
)

func main() {
	e := echo.New()

	swagger, err := openapi.GetSwagger()
	if err != nil {
		e.Logger.Fatal("Error loading swagger spec\n: %s", err)
	}

	baseURL := "/api"
	swagger.Servers = openapi3.Servers{&openapi3.Server{URL: baseURL}}

	// middlewares
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:8080", "https://*.traq-preview.trapti.tech", "https://*.livekit.trap.show"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
	}))
	//e.Use(oapimiddleware.OapiRequestValidator(swagger))

	// connect to database
	db, err := sqlx.Connect("mysql", config.MySQL().FormatDSN())
	if err != nil {
		e.Logger.Fatal(err)
	}
	defer db.Close()

	// migrate tables
	if err := migration.MigrateTables(db.DB); err != nil {
		e.Logger.Fatal(err)
	}

	// setup repository
	livekitConfig := config.LoadLivekitConfig()
	repo := repository.New(db, livekitConfig)
	if err = repo.InitializeRoomState(); err != nil {
		e.Logger.Fatal("Failed to initialize room state: %v", err)
	}

	// setup routes
	cfg := config.NewS3Config()
	fileSvc := repository.NewFileService(cfg)
	h := handler.New(repo, fileSvc)
	openapi.RegisterHandlersWithBaseURL(e, h, baseURL)

	e.Logger.Fatal(e.Start(config.AppAddr()))
}
