// This package provides the entrypoint for the module
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	_ "net/http/pprof"

	"github.com/viam-modules/viamrtsp"
	"go.uber.org/zap/zapcore"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/utils"
)

func main() {
	go func() {
		fmt.Println("sanity checking...")
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("viamrtsp"))
}

func mainWithArgs(ctx context.Context, _ []string, logger logging.Logger) error {
	myMod, err := module.NewModuleFromArgs(ctx)
	if err != nil {
		return err
	}

	if logger.Level() != zapcore.DebugLevel {
		logger.Info("suppressing non fatal libav errors / warnings due to false positives. to unsuppress, set module log_level to 'debug'")
		viamrtsp.SetLibAVLogLevelFatal()
	}

	for _, model := range viamrtsp.Models {
		err = myMod.AddModelFromRegistry(ctx, camera.API, model)
		if err != nil {
			return err
		}
	}

	err = myMod.Start(ctx)
	defer myMod.Close(ctx)
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}
