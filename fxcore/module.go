package fxcore

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	gommonlog "github.com/labstack/gommon/log"
	"github.com/templatedop/ftptemplate/config"
	"github.com/templatedop/ftptemplate/fxconfig"
	"github.com/templatedop/ftptemplate/fxgenerate"
	"github.com/templatedop/ftptemplate/fxhealthcheck"
	"github.com/templatedop/ftptemplate/fxlog"
	"github.com/templatedop/ftptemplate/generate/uuid"
	"github.com/templatedop/ftptemplate/healthcheck"
	"github.com/templatedop/ftptemplate/httpserver"
	httpservermiddleware "github.com/templatedop/ftptemplate/httpserver/middleware"
	"github.com/templatedop/ftptemplate/log"
	"go.uber.org/fx"
)

const (
	ModuleName                      = "core"
	DefaultAddress                  = ":8081"
	DefaultMetricsPath              = "/metrics"
	DefaultHealthCheckStartupPath   = "/healthz"
	DefaultHealthCheckLivenessPath  = "/livez"
	DefaultHealthCheckReadinessPath = "/readyz"
	DefaultDebugConfigPath          = "/debug/config"
	DefaultDebugPProfPath           = "/debug/pprof"
	DefaultDebugBuildPath           = "/debug/build"
	DefaultDebugRoutesPath          = "/debug/routes"
	DefaultDebugStatsPath           = "/debug/stats"
	DefaultDebugModulesPath         = "/debug/modules"
	ThemeLight                      = "light"
	ThemeDark                       = "dark"
)

var FxCoreModule = fx.Module(
	ModuleName,
	fxgenerate.FxGenerateModule,
	fxconfig.FxConfigModule,
	fxlog.FxLogModule,

	fxhealthcheck.FxHealthcheckModule,
	fx.Provide(
		NewFxModuleInfoRegistry,
		NewFxCore,
		fx.Annotate(
			NewFxCoreModuleInfo,
			fx.As(new(interface{})),
			fx.ResultTags(`group:"core-module-infos"`),
		),
	),
	fx.Invoke(func(logger *log.Logger, core *Core) {
		logger.Debug().Msg("starting core")
	}),
)

type FxCoreDashboardTheme struct {
	Theme string `form:"theme" json:"theme"`
}

type FxCoreParam struct {
	fx.In
	Context   context.Context
	LifeCycle fx.Lifecycle
	Generator uuid.UuidGenerator

	Checker  *healthcheck.Checker
	Config   *config.Config
	Logger   *log.Logger
	Registry *FxModuleInfoRegistry
}

// NewFxCore returns a new [Core].
func NewFxCore(p FxCoreParam) (*Core, error) {
	appDebug := p.Config.AppDebug()

	// logger
	coreLogger := httpserver.NewEchoLogger(
		log.FromZerolog(p.Logger.ToZerolog().With().Str("module", ModuleName).Logger()),
	)

	coreServer, err := httpserver.NewDefaultHttpServerFactory().Create(
		httpserver.WithDebug(appDebug),
		httpserver.WithBanner(false),
		httpserver.WithLogger(coreLogger),
		httpserver.WithHttpErrorHandler(
			httpserver.JsonErrorHandler(
				p.Config.GetBool("modules.core.server.errors.obfuscate") || !appDebug,
				p.Config.GetBool("modules.core.server.errors.stack") || appDebug,
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create core http server: %w", err)
	}

	// middlewares
	coreServer = withMiddlewares(coreServer, p)

	// lifecycles
	p.LifeCycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			address := p.Config.GetString("modules.core.server.address")
			if address == "" {
				address = DefaultAddress
			}

			go coreServer.Start(address)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return coreServer.Shutdown(ctx)
		},
	})

	return NewCore(p.Config, p.Checker, coreServer), nil
}

func withMiddlewares(coreServer *echo.Echo, p FxCoreParam) *echo.Echo {
	// CORS middleware
	coreServer.Use(middleware.CORS())

	// request id middleware
	coreServer.Use(httpservermiddleware.RequestIdMiddlewareWithConfig(
		httpservermiddleware.RequestIdMiddlewareConfig{
			Generator: p.Generator,
		},
	))

	// request logger middleware
	requestHeadersToLog := map[string]string{
		httpservermiddleware.HeaderXRequestId: httpservermiddleware.LogFieldRequestId,
	}

	for headerName, fieldName := range p.Config.GetStringMapString("modules.core.server.log.headers") {
		requestHeadersToLog[headerName] = fieldName
	}

	coreServer.Use(httpservermiddleware.RequestLoggerMiddlewareWithConfig(
		httpservermiddleware.RequestLoggerMiddlewareConfig{
			RequestHeadersToLog:             requestHeadersToLog,
			RequestUriPrefixesToExclude:     p.Config.GetStringSlice("modules.core.server.log.exclude"),
			LogLevelFromResponseOrErrorCode: p.Config.GetBool("modules.core.server.log.level_from_response"),
		},
	))

	// recovery middleware
	coreServer.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableErrorHandler: true,
		LogLevel:            gommonlog.ERROR,
	}))

	return coreServer
}
