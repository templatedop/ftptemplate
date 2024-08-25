package fxdb

import (
	"context"

	"github.com/templatedop/ftptemplate/db"
	"github.com/templatedop/ftptemplate/log"

	"github.com/templatedop/ftptemplate/config"
	"go.uber.org/fx"
)

const (
	ModuleName = "database"
)

var FxDBModule = fx.Module(ModuleName,
	fx.Provide(db.NewDBConfig, fx.Private),
	fx.Provide(
		//NewDBConfig ,
		db.Pgxconfig,
		db.NewDB,
	),

	fx.Invoke(func(db *db.DB, log *log.Logger, c *config.Config, lc fx.Lifecycle) error {

		log.Debug().Str("module", ModuleName).Msg("Invoking fxdb module")

		lc.Append(fx.Hook{

			OnStart: func(ctx context.Context) error {
				log.Debug().Str("module", ModuleName).Msg("Starting fxdb module")
				//log.Debug("Inside fxdb/fx.go")
				err := db.Ping(ctx)
				if err != nil {
					return err
				}
				log.Info().Msg("Successfully connected to the database")
				
				return nil
			},
			OnStop: func(ctx context.Context) error {
				db.Close()
				return nil
			},
		})
		return nil
	}),
)
