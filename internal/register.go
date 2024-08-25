package internal

import (
	"github.com/templatedop/ftptemplate/fxcron"
	"github.com/templatedop/ftptemplate/internal/cron"
	"go.uber.org/fx"
)

func Register() fx.Option {
	return fx.Options(
		fxcron.AsCronJob(
			cron.NewExampleCronJob, // register the ExampleCronJob
			`*/2 * * * *`,          // to run every 2 minutes
			// gocron.WithLimitedRuns(10),    // and with 10 max runs
		),
		fxcron.AsCronJob(
			cron.OneNewExampleCronJob, // register the ExampleCronJob
			`*/2 * * * *`,             // to run every 2 minutes
			// gocron.WithLimitedRuns(10),    // and with 10 max runs
		),
	)
}
