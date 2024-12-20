// This package provides the entrypoint for the module
package main

import (
	"context"

	_ "net/http/pprof"

	"github.com/viam-modules/viamrtsp"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/module"
)

func main() {
	err := realMain(context.Background())
	if err != nil {
		panic(err)
	}
}

func realMain(ctx context.Context) error {
	myMod, err := module.NewModuleFromArgs(ctx)
	if err != nil {
		return err
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
