package fxconfig

import (
	"os"

	"github.com/templatedop/ftptemplate/config"

	"go.uber.org/fx"
)

const ModuleName = "config"

var FxConfigModule = fx.Module(
	ModuleName,
	fx.Provide(
		config.NewDefaultConfigFactory,
		NewFxConfig,
	),
)

type FxConfigParam struct {
	fx.In
	Factory config.ConfigFactory
}

func NewFxConfig(p FxConfigParam) (*config.Config, error) {
	return p.Factory.Create(
		config.WithFileName("config"),
		config.WithFilePaths(
			".",
			"./configs",
			os.Getenv("APP_CONFIG_PATH"),
		),
	)
}
