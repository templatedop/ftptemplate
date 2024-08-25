package fxcore

import (
	"github.com/rs/zerolog"
	"github.com/templatedop/ftptemplate/config"
	"github.com/templatedop/ftptemplate/log"
	"go.uber.org/fx"
)

type FxExtraInfo interface {
	Name() string
	Value() string
}

type fxExtraInfo struct {
	name  string
	value string
}

func NewFxExtraInfo(name string, value string) FxExtraInfo {
	return &fxExtraInfo{
		name:  name,
		value: value,
	}
}

func (i *fxExtraInfo) Name() string {
	return i.name
}

func (i *fxExtraInfo) Value() string {
	return i.value
}

type FxModuleInfo interface {
	Name() string
	Data() map[string]any
}

type FxCoreModuleInfo struct {
	AppName    string
	AppEnv     string
	AppDebug   bool
	AppVersion string
	LogLevel   string
	LogOutput  string

	ExtraInfos map[string]string
}

type FxCoreModuleInfoParam struct {
	fx.In
	Config     *config.Config
	ExtraInfos []FxExtraInfo `group:"core-extra-infos"`
}

func NewFxCoreModuleInfo(p FxCoreModuleInfoParam) *FxCoreModuleInfo {
	logLevel, logOutput := "", ""
	if p.Config.IsTestEnv() {
		logLevel = zerolog.DebugLevel.String()
		logOutput = log.TestOutputWriter.String()
	} else {
		logLevel = log.FetchLogLevel(p.Config.GetString("modules.log.level")).String()
		logOutput = log.FetchLogOutputWriter(p.Config.GetString("modules.log.output")).String()
	}

	extraInfos := make(map[string]string)
	for _, info := range p.ExtraInfos {
		extraInfos[info.Name()] = info.Value()
	}

	return &FxCoreModuleInfo{
		AppName:    p.Config.AppName(),
		AppEnv:     p.Config.AppEnv(),
		AppDebug:   p.Config.AppDebug(),
		AppVersion: p.Config.AppVersion(),
		LogLevel:   logLevel,
		LogOutput:  logOutput,

		ExtraInfos: extraInfos,
	}
}

// Name return the name of the module info.
func (i *FxCoreModuleInfo) Name() string {
	return ModuleName
}

// Data return the data of the module info.
func (i *FxCoreModuleInfo) Data() map[string]interface{} {
	return map[string]interface{}{
		"app": map[string]interface{}{
			"name":    i.AppName,
			"env":     i.AppEnv,
			"debug":   i.AppDebug,
			"version": i.AppVersion,
		},
		"log": map[string]interface{}{
			"level":  i.LogLevel,
			"output": i.LogOutput,
		},

		"extra": i.ExtraInfos,
	}
}
