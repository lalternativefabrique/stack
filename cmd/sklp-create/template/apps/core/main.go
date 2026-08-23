// @title           Core API
// @version         0.1.0
// @description     Stack application core REST API.
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Pass `Bearer <token>`. JWT is verified by the gateway; the user id arrives as the X-User-Id header.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
	"github.com/lalternative/packages/go/eda/pkg/consumer"
	"github.com/nats-io/nats.go"

	"app/core/account"
	"app/core/example"
	exampleeventhandlers "app/core/example/application/event-handlers"
	"app/core/health"
	"app/core/middleware"
	"app/core/observability"
	"app/core/pkg/db"
)

func main() {
	ctx := context.Background()

	shutdown, err := observability.Init(ctx, observability.Config{
		ServiceName: "core",
		Endpoint:    os.Getenv("SKALPAI_ENDPOINT"),
		APIKey:      os.Getenv("SKALPAI_API_KEY"),
		ProjectID:   os.Getenv("SKALPAI_PROJECT_ID"),
	})
	if err != nil {
		log.Printf("observability init: %v (continuing without telemetry)", err)
	}
	defer shutdown(ctx)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := db.Open(ctx, dbURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, "./migrations/postgres"); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Drain()

	// Durable JetStream consumers. consumer.Run owns all redelivery semantics
	// (Term on permanent errors, bounded MaxDeliver, staged BackOff, the DLQ
	// stream, ack heartbeats, reconnect). Register one goroutine per handler;
	// add yours under apps/core/<context>/application/event-handlers/ — never hand-roll a
	// JetStream subscription.
	consumerCtx, stopConsumers := context.WithCancel(ctx)
	defer stopConsumers()
	go consumer.Run(consumerCtx, nc, exampleeventhandlers.NewExampleCreatedHandler(), consumer.Config{})

	e := echo.New()
	e.HideBanner = true
	e.Use(emw.Recover(), emw.RequestID(), observability.EchoMiddleware("core"))

	health.Register(e)

	// Account/auth endpoints (e.g. logout) are public: clearing the session
	// cookie must work without a valid token, so they mount on the root Echo
	// instance rather than under the RequireAuth-guarded /api/v1 group.
	account.NewService(pool).RegisterRoutes(e)

	// Protected API. The web app reverse-proxies browser requests to /api/v1/*
	// verbatim (see apps/web routes/api/v1/$.ts). Every request carries the
	// `token` cookie (or Authorization: Bearer) verified by RequireAuth.
	protected := e.Group("/api/v1", middleware.RequireAuth())
	protected.GET("/me", func(c echo.Context) error {
		u, _ := middleware.GetUser(c)
		return c.JSON(http.StatusOK, map[string]string{
			"user_id": u.ID,
			"email":   u.Email,
			"name":    u.Name,
		})
	})
	exampleSvc, err := example.NewService(ctx)
	if err != nil {
		log.Fatalf("example service: %v", err)
	}
	exampleSvc.RegisterRoutes(protected)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4100"
	}
	log.Printf("core listening on :%s (nats=%s)", port, natsURL)
	go func() {
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Block on SIGTERM rather than on Start: the deferred cleanups above
	// (consumers, NATS drain, pool) never run past a log.Fatal, which exits
	// the process outright. Kubernetes sends SIGTERM and waits
	// terminationGracePeriodSeconds before SIGKILL, so keep the drain under it.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Print("shutting down")

	drainCtx, cancelDrain := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelDrain()
	if err := e.Shutdown(drainCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
}
