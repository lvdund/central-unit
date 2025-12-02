package app

import (
	"context"
	"fmt"

	"central-unit/internal/common/logger"
	cucontext "central-unit/internal/context"
	"central-unit/pkg/config"
	"central-unit/pkg/model"
)

// App represents the CU-CP application
type App struct {
	cfg    config.Config
	logger *logger.Logger
	cuCtx  *cucontext.CuCpContext
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new App instance
// It loads the config internally before initializing the application
func New(cfgPath string) (*App, error) {
	// Load config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize logger
	log := logger.InitLogger(cfg.Logging.Level, map[string]string{"mod": "app"})
	logger.ParseLogLevel(cfg.Logging.Level)

	// Create app instance
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		cfg:    cfg,
		logger: log,
		ctx:    ctx,
		cancel: cancel,
	}

	return app, nil
}

// Start starts the CU-CP application
func (a *App) Start() error {
	a.logger.Info("Starting CU-CP application %s", a.cfg.CUCP.NodeName)

	// Initialize CU-CP context with config
	// Convert config to model.AMF for initialization
	amf := model.AMF{
		Ip:   a.cfg.NGAP.AMFAddress,
		Port: a.cfg.NGAP.AMFPort,
	}

	// Initialize CU-CP context
	cuCtx := cucontext.InitContext(amf, a.cfg)

	// Update context
	cuCtx.Ctx = a.ctx
	a.cuCtx = cuCtx

	a.logger.Info("CU-CP application started successfully")
	return nil
}

// Stop stops the CU-CP application gracefully
func (a *App) Stop(ctx context.Context) error {
	a.logger.Info("Stopping CU-CP application")

	// Cancel context to signal shutdown
	if a.cancel != nil {
		a.cancel()
	}

	// Terminate CU-CP context if initialized
	if a.cuCtx != nil {
		a.cuCtx.Terminate()
	}

	a.logger.Info("CU-CP application stopped")
	return nil
}
