package fxcron

import (
	"context"
	
	"time"

	"github.com/templatedop/ftptemplate/config"
	"github.com/templatedop/ftptemplate/generate/uuid"
	"github.com/templatedop/ftptemplate/log"	
	"github.com/go-co-op/gocron/v2"	
	"go.uber.org/fx"
)

const (
	ModuleName                           = "cron"
	LogRecordFieldCronJobName            = "cronJob"
	LogRecordFieldCronJobExecutionId     = "cronJobExecutionID"
	TraceSpanAttributeCronJobName        = "CronJob"
	TraceSpanAttributeCronJobExecutionId = "CronJobExecutionID"
)


var FxCronModule = fx.Module(
	ModuleName,
	fx.Provide(
		NewDefaultCronSchedulerFactory,
		NewFxCronJobRegistry,
		NewFxCron,
		fx.Annotate(
			NewFxCronModuleInfo,
			fx.As(new(interface{})),
			fx.ResultTags(`group:"core-module-infos"`),
		),
	),
)

// FxCronParam allows injection of the required dependencies in [NewFxCron].
type FxCronParam struct {
	fx.In
	LifeCycle       fx.Lifecycle
	Generator       uuid.UuidGenerator
	
	Factory         CronSchedulerFactory
	Config          *config.Config
	Registry        *CronJobRegistry
	Logger          *log.Logger
	
}

// NewFxCron returns a new [gocron.Scheduler].
//
//nolint:cyclop,gocognit
func NewFxCron(p FxCronParam) (gocron.Scheduler, error) {
	appDebug := p.Config.AppDebug()

	// logger
	cronLogger := log.FromZerolog(p.Logger.ToZerolog().With().Str("system", ModuleName).Logger())

	

	// scheduler
	cronSchedulerOptions, err := buildSchedulerOptions(p.Config)
	if err != nil {
		p.Logger.Error().Err(err).Msg("cron scheduler options creation error")

		return nil, err
	}

	cronScheduler, err := p.Factory.Create(cronSchedulerOptions...)
	if err != nil {
		p.Logger.Error().Err(err).Msg("cron scheduler creation error")

		return nil, err
	}

	// jobs logs
	cronJobLogExecution := p.Config.GetBool("modules.cron.log.enabled") || appDebug
	cronJobLogExclusions := p.Config.GetStringSlice("modules.cron.log.exclude")

	
	// jobs registration
	cronJobs, err := p.Registry.ResolveCronJobs()
	if err != nil {
		p.Logger.Error().Err(err).Msg("cron jobs resolution error")

		return nil, err
	}

	for _, cronJob := range cronJobs {
		// var scoping
		currentCronJob := cronJob

		currentCronJobName := currentCronJob.Implementation().Name()
		currentJobOptions := append(currentCronJob.Options(), gocron.WithName(currentCronJobName))
		currentCronJobLogExecution := !Contains(cronJobLogExclusions, currentCronJobName)
		

		_, err = cronScheduler.NewJob(
			gocron.CronJob(
				currentCronJob.Expression(),
				p.Config.GetBool("modules.cron.scheduler.seconds"),
			),
			gocron.NewTask(
				func() {
					currentCronJobExecutionId := p.Generator.Generate()

					currentCronJobCtx := context.WithValue(context.Background(), CtxCronJobNameKey{}, currentCronJobName)
					currentCronJobCtx = context.WithValue(currentCronJobCtx, CtxCronJobExecutionIdKey{}, currentCronJobExecutionId)
					

					

					currentCronJobLogger := log.FromZerolog(
						cronLogger.
							ToZerolog().
							With().
							Str(LogRecordFieldCronJobName, currentCronJobName).
							Str(LogRecordFieldCronJobExecutionId, currentCronJobExecutionId).
							Logger(),
					)

					currentCronJobCtx = currentCronJobLogger.WithContext(currentCronJobCtx)

					

					if cronJobLogExecution && currentCronJobLogExecution {
						currentCronJobLogger.Info().Msg("job execution start")
					}

					runErr := currentCronJob.Implementation().Run(currentCronJobCtx)

					if runErr != nil {
						
						currentCronJobLogger.Error().Err(runErr).Msg("job execution error")
					} else {
						

						if cronJobLogExecution && currentCronJobLogExecution {
							currentCronJobLogger.Info().Msg("job execution success")
						}
					}
				},
			),
			currentJobOptions...,
		)

		if err != nil {
			cronLogger.Error().Err(err).Msgf("job registration error for job %s with %s", currentCronJobName, currentCronJob.Expression())

			return nil, err
		} else {
			cronLogger.Debug().Msgf("job registration success for job %s with %s", currentCronJobName, currentCronJob.Expression())
		}
	}

	// lifecycles
	p.LifeCycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			cronLogger.Debug().Msg("starting cron scheduler")

			cronScheduler.Start()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			cronLogger.Debug().Msg("stopping cron scheduler")

			return cronScheduler.Shutdown()
		},
	})

	return cronScheduler, nil
}

//nolint:cyclop
func buildSchedulerOptions(cfg *config.Config) ([]gocron.SchedulerOption, error) {
	var options []gocron.SchedulerOption

	// location, default local
	if cfgLocation := cfg.GetString("modules.cron.scheduler.location"); cfgLocation != "" {
		location, err := time.LoadLocation(cfgLocation)
		if err != nil {
			return nil, err
		}

		options = append(options, gocron.WithLocation(location))
	}

	// concurrency
	if cfg.GetBool("modules.cron.scheduler.concurrency.limit.enabled") {
		var mode gocron.LimitMode
		if cfg.GetString("modules.cron.scheduler.concurrency.limit.mode") == "reschedule" {
			mode = gocron.LimitModeReschedule
		} else {
			mode = gocron.LimitModeWait
		}

		options = append(options, gocron.WithLimitConcurrentJobs(cfg.GetUint("modules.cron.scheduler.concurrency.limit.max"), mode))
	}

	// stop timeout, default 10s
	if cfgStopTimeout := cfg.GetString("modules.cron.scheduler.stop.timeout"); cfgStopTimeout != "" {
		stopTimeout, err := time.ParseDuration(cfgStopTimeout)
		if err != nil {
			return nil, err
		}

		options = append(options, gocron.WithStopTimeout(stopTimeout))
	}

	// jobs global options
	var jobsOptions []gocron.JobOption

	// jobs execution start
	if cfg.GetBool("modules.cron.jobs.execution.start.immediately") {
		jobsOptions = append(jobsOptions, gocron.WithStartAt(gocron.WithStartImmediately()))
	} else if cfgJobsStartAt := cfg.GetString("modules.cron.jobs.execution.start.at"); cfgJobsStartAt != "" {
		jobsStartAt, err := time.Parse(time.RFC3339, cfgJobsStartAt)
		if err != nil {
			return nil, err
		}

		jobsOptions = append(jobsOptions, gocron.WithStartAt(gocron.WithStartDateTime(jobsStartAt)))
	}

	// jobs execution limit
	if cfg.GetBool("modules.cron.jobs.execution.limit.enabled") {
		jobsOptions = append(jobsOptions, gocron.WithLimitedRuns(cfg.GetUint("modules.cron.jobs.execution.limit.max")))
	}

	// jobs execution mode
	if cfg.GetBool("modules.cron.jobs.singleton.enabled") {
		var mode gocron.LimitMode
		if cfg.GetString("modules.cron.jobs.singleton.mode") == "reschedule" {
			mode = gocron.LimitModeReschedule
		} else {
			mode = gocron.LimitModeWait
		}
		jobsOptions = append(jobsOptions, gocron.WithSingletonMode(mode))
	}

	if len(jobsOptions) > 0 {
		options = append(options, gocron.WithGlobalJobOptions(jobsOptions...))
	}

	return options, nil
}
