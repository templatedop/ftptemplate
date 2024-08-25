package config

import (
	"os"
	"github.com/spf13/viper"
)

const (
	AppEnvProd        = "prod"    // prod environment
	AppEnvDev         = "dev"     // dev environment
	AppEnvTest        = "test"    // test environment
	DefaultAppName    = "app"     // default application name
	DefaultAppVersion = "unknown" // default application version
)

type Config struct {
	*viper.Viper
}

func (c *Config) GetEnvVar(envVar string) string {
	return os.Getenv(envVar)
}

func (c *Config) AppName() string {
	return c.GetString("app.name")
}

func (c *Config) AppEnv() string {
	return c.GetString("app.env")
}

func (c *Config) AppVersion() string {
	return c.GetString("app.version")
}

func (c *Config) AppDebug() bool {
	return c.GetBool("app.debug")
}

func (c *Config) IsProdEnv() bool {
	return c.AppEnv() == AppEnvProd
}

func (c *Config) IsDevEnv() bool {
	return c.AppEnv() == AppEnvDev
}

func (c *Config) IsTestEnv() bool {
	return c.AppEnv() == AppEnvTest
}
