package fxcore

import (
	"context"
	"testing"

	"github.com/templatedop/ftptemplate/fxlog"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type Bootstrapper struct {
	context context.Context
	options []fx.Option
}

func NewBootstrapper() *Bootstrapper {
	return &Bootstrapper{
		context: context.Background(),
		options: []fx.Option{
			FxCoreModule,
		},
	}
}

func (b *Bootstrapper) WithContext(ctx context.Context) *Bootstrapper {
	b.context = ctx

	return b
}

func (b *Bootstrapper) WithOptions(options ...fx.Option) *Bootstrapper {
	b.options = append(b.options, options...)

	return b
}

func (b *Bootstrapper) BootstrapApp(options ...fx.Option) *fx.App {
	return fx.New(
		fx.Supply(fx.Annotate(b.context, fx.As(new(context.Context)))),
		fx.WithLogger(fxlog.NewFxEventLogger),
		fx.Options(b.options...),
		fx.Options(options...),
	)
}

func (b *Bootstrapper) BootstrapTestApp(tb testing.TB, options ...fx.Option) *fxtest.App {
	tb.Helper()

	tb.Setenv("APP_ENV", "test")

	return fxtest.New(
		tb,
		fx.Supply(fx.Annotate(b.context, fx.As(new(context.Context)))),
		fx.NopLogger,
		fx.Options(b.options...),
		fx.Options(options...),
	)
}

func (b *Bootstrapper) RunApp(options ...fx.Option) {
	b.BootstrapApp(options...).Run()
}

func (b *Bootstrapper) RunTestApp(tb testing.TB, options ...fx.Option) {
	tb.Helper()

	b.BootstrapTestApp(tb, options...).RequireStart().RequireStop()
}
