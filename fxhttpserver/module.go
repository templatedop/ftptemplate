package fxhttpserver

import (
	"context"
	"fmt"
	
	"strings"

	"github.com/templatedop/ftptemplate/config"
	"github.com/templatedop/ftptemplate/generate/uuid"
	"github.com/templatedop/ftptemplate/httpserver"
	httpservermiddleware "github.com/templatedop/ftptemplate/httpserver/middleware"
	"github.com/templatedop/ftptemplate/log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	gommonlog "github.com/labstack/gommon/log"
	
	"go.uber.org/fx"
)

const (
	ModuleName     = "httpserver"
	DefaultAddress = ":8080"
)


var FxHttpServerModule = fx.Module(
	ModuleName,
	fx.Provide(
		httpserver.NewDefaultHttpServerFactory,
		NewFxHttpServerRegistry,
		NewFxHttpServer,
		fx.Annotate(
			NewFxHttpServerModuleInfo,
			fx.As(new(interface{})),
			fx.ResultTags(`group:"core-module-infos"`),
		),
	),
)

// FxHttpServerParam allows injection of the required dependencies in [NewFxHttpServer].
type FxHttpServerParam struct {
	fx.In
	LifeCycle       fx.Lifecycle
	Factory         httpserver.HttpServerFactory
	Generator       uuid.UuidGenerator
	Registry        *HttpServerRegistry
	Config          *config.Config
	Logger          *log.Logger
	
}

// NewFxHttpServer returns a new [echo.Echo].
func NewFxHttpServer(p FxHttpServerParam) (*echo.Echo, error) {
	appDebug := p.Config.AppDebug()

	// logger
	echoLogger := httpserver.NewEchoLogger(
		log.FromZerolog(p.Logger.ToZerolog().With().Str("module", ModuleName).Logger()),
	)

	// renderer
	var renderer echo.Renderer
	

	// server
	httpServer, err := p.Factory.Create(
		httpserver.WithDebug(appDebug),
		httpserver.WithBanner(false),
		httpserver.WithLogger(echoLogger),
		httpserver.WithRenderer(renderer),
		httpserver.WithHttpErrorHandler(
			httpserver.JsonErrorHandler(
				p.Config.GetBool("modules.http.server.errors.obfuscate") || !appDebug,
				p.Config.GetBool("modules.http.server.errors.stack") || appDebug,
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http server: %w", err)
	}

	// middlewares
	httpServer = withDefaultMiddlewares(httpServer, p)

	// groups, handlers & middlewares registrations
	httpServer = withRegisteredResources(httpServer, p)

	// lifecycles
	p.LifeCycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !p.Config.IsTestEnv() {
				address := p.Config.GetString("modules.http.server.address")
				if address == "" {
					address = DefaultAddress
				}

				//nolint:errcheck
				go httpServer.Start(address)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if !p.Config.IsTestEnv() {
				return httpServer.Shutdown(ctx)
			}

			return nil
		},
	})

	return httpServer, nil
}

func withDefaultMiddlewares(httpServer *echo.Echo, p FxHttpServerParam) *echo.Echo {
	// request id middleware
	httpServer.Use(httpservermiddleware.RequestIdMiddlewareWithConfig(
		httpservermiddleware.RequestIdMiddlewareConfig{
			Generator: p.Generator,
		},
	))

	

	// request logger middleware
	requestHeadersToLog := map[string]string{
		httpservermiddleware.HeaderXRequestId: httpservermiddleware.LogFieldRequestId,
	}

	for headerName, fieldName := range p.Config.GetStringMapString("modules.http.server.log.headers") {
		requestHeadersToLog[headerName] = fieldName
	}

	httpServer.Use(httpservermiddleware.RequestLoggerMiddlewareWithConfig(
		httpservermiddleware.RequestLoggerMiddlewareConfig{
			RequestHeadersToLog:             requestHeadersToLog,
			RequestUriPrefixesToExclude:     p.Config.GetStringSlice("modules.http.server.log.exclude"),
			LogLevelFromResponseOrErrorCode: p.Config.GetBool("modules.http.server.log.level_from_response"),
		},
	))



	// recovery middleware
	httpServer.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableErrorHandler: true,
		LogLevel:            gommonlog.ERROR,
	}))

	return httpServer
}

func withRegisteredResources(httpServer *echo.Echo, p FxHttpServerParam) *echo.Echo {
	// register handler groups
	resolvedHandlersGroups, err := p.Registry.ResolveHandlersGroups()
	if err != nil {
		httpServer.Logger.Errorf("cannot resolve router handlers groups: %v", err)
	}

	for _, g := range resolvedHandlersGroups {
		group := httpServer.Group(g.Prefix(), g.Middlewares()...)

		for _, h := range g.Handlers() {
			group.Add(
				strings.ToUpper(h.Method()),
				h.Path(),
				h.Handler(),
				h.Middlewares()...,
			)
			httpServer.Logger.Debugf("registering handler in group for [%s] %s%s", h.Method(), g.Prefix(), h.Path())
		}

		httpServer.Logger.Debugf("registered handlers group for prefix %s", g.Prefix())
	}

	// register middlewares
	resolvedMiddlewares, err := p.Registry.ResolveMiddlewares()
	if err != nil {
		httpServer.Logger.Errorf("cannot resolve router middlewares: %v", err)
	}

	for _, m := range resolvedMiddlewares {
		if m.Kind() == GlobalPre {
			httpServer.Pre(m.Middleware())
		}

		if m.Kind() == GlobalUse {
			httpServer.Use(m.Middleware())
		}

		httpServer.Logger.Debugf("registered %s middleware %T", m.Kind().String(), m.Middleware())
	}

	// register handlers
	resolvedHandlers, err := p.Registry.ResolveHandlers()
	if err != nil {
		httpServer.Logger.Errorf("cannot resolve router handlers: %v", err)
	}

	for _, h := range resolvedHandlers {
		httpServer.Add(
			strings.ToUpper(h.Method()),
			h.Path(),
			h.Handler(),
			h.Middlewares()...,
		)

		httpServer.Logger.Debugf("registered handler for [%s] %s", h.Method(), h.Path())
	}

	return httpServer
}
