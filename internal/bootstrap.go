package internal

import (
	"github.com/templatedop/ftptemplate/fxdb"
	"github.com/templatedop/ftptemplate/fxcore"
	"github.com/templatedop/ftptemplate/fxcron"
)

var Bootstrapper = fxcore.NewBootstrapper().WithOptions(
	fxdb.FxDBModule,
	fxcron.FxCronModule,
	Register(),
)
