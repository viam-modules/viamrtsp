// This package provides the entrypoint for the module
package main

import (
	"context"
	"errors"
	"os"

	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/garmin"
	"github.com/viam-modules/viamrtsp/ptzclient"
	"github.com/viam-modules/viamrtsp/unifi"
	"github.com/viam-modules/viamrtsp/upnpdiscovery"
	"github.com/viam-modules/viamrtsp/viamonvif"
	"github.com/viam-modules/viamrtsp/videostore"
	vsutils "github.com/viam-modules/video-store/videostore/utils"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/rdk/services/video"
	"go.viam.com/utils"
)

func main() {
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("viamrtsp"))
}

func mainWithArgs(ctx context.Context, _ []string, logger logging.Logger) error {
	if logger.GetLevel() == logging.DEBUG {
		vsutils.SetLibAVLogLevel("debug")
	} else {
		vsutils.SetLibAVLogLevel("fatal")
	}
	vsutils.SetFFmpegLogCallback(logger)

	// NewModule, not NewModuleFromArgs: latter spawns its own moduleLogger, stranding our ffmpeg sublogger on stdout fallback.
	if len(os.Args) < 2 { //nolint:mnd
		return errors.New("need socket path as command line argument")
	}
	myMod, err := module.NewModule(ctx, os.Args[1], logger)
	if err != nil {
		return err
	}

	for _, model := range viamrtsp.Models {
		err = myMod.AddModelFromRegistry(ctx, camera.API, model)
		if err != nil {
			return err
		}
	}

	err = myMod.AddModelFromRegistry(ctx, generic.API, videostore.ComponentModel)
	if err != nil {
		return err
	}

	err = myMod.AddModelFromRegistry(ctx, video.API, videostore.ServiceModel)
	if err != nil {
		return err
	}

	err = myMod.AddModelFromRegistry(ctx, discovery.API, viamonvif.Model)
	if err != nil {
		return err
	}
	err = myMod.AddModelFromRegistry(ctx, discovery.API, upnpdiscovery.Model)
	if err != nil {
		return err
	}
	err = myMod.AddModelFromRegistry(ctx, discovery.API, garmin.Model)
	if err != nil {
		return err
	}
	err = myMod.AddModelFromRegistry(ctx, generic.API, ptzclient.Model)
	if err != nil {
		return err
	}

	err = myMod.AddModelFromRegistry(ctx, discovery.API, unifi.Model)
	if err != nil {
		return err
	}

	err = myMod.Start(ctx)
	defer myMod.Close(ctx)
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}
