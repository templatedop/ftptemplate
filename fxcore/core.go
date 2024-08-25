package fxcore

import (
	"github.com/templatedop/ftptemplate/config"
	"github.com/templatedop/ftptemplate/healthcheck"
	"github.com/labstack/echo/v4"
)


type Core struct {
	config     *config.Config
	checker    *healthcheck.Checker
	httpServer *echo.Echo
}


func NewCore(config *config.Config, checker *healthcheck.Checker, httpServer *echo.Echo) *Core {
	return &Core{
		config:     config,
		checker:    checker,
		httpServer: httpServer,
	}
}

func (c *Core) Config() *config.Config {
	return c.config
}


func (c *Core) Checker() *healthcheck.Checker {
	return c.checker
}


func (c *Core) HttpServer() *echo.Echo {
	return c.httpServer
}
