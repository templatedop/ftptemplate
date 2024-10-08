package fxlog

import (
	"io"
	"os"

	"github.com/templatedop/ftptemplate/config"
	"github.com/templatedop/ftptemplate/log"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const ModuleName = "log"

var FxLogModule = fx.Module(
	ModuleName,
	fx.Provide(
		log.NewDefaultLoggerFactory,
		NewFxLogger,
	),
)

// FxLogParam allows injection of the required dependencies in [NewFxLogger].
type FxLogParam struct {
	fx.In
	Factory log.LoggerFactory
	Config  *config.Config
}

// NewFxLogger returns a [log.Logger].
func NewFxLogger(p FxLogParam) (*log.Logger, error) {
	var level zerolog.Level
	if p.Config.AppDebug() {
		level = zerolog.DebugLevel
	} else {
		level = log.FetchLogLevel(p.Config.GetString("modules.log.level"))
	}

	var outputWriter io.Writer

	switch log.FetchLogOutputWriter(p.Config.GetString("modules.log.output")) {
	case log.NoopOutputWriter:
		outputWriter = io.Discard
	case log.ConsoleOutputWriter:
		outputWriter = zerolog.ConsoleWriter{Out: os.Stderr}
	default:
		outputWriter = os.Stdout
	}

	return p.Factory.Create(
		log.WithServiceName(p.Config.AppName()),
		log.WithLevel(level),
		log.WithOutputWriter(outputWriter),
	)
}
