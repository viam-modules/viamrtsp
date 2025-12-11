// This package provides the entrypoint for the module
package main

import (
	"context"

	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/ptzclient"
	"github.com/viam-modules/viamrtsp/unifi"
	"github.com/viam-modules/viamrtsp/upnpdiscovery"
	"github.com/viam-modules/viamrtsp/viamonvif"
	"github.com/viam-modules/viamrtsp/videostore"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/rdk/services/video"
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
	err = myMod.AddModelFromRegistry(ctx, generic.API, ptzclient.Model)
	if err != nil {
		return err
	}

	err = myMod.AddModelFromRegistry(ctx, discovery.API, unifi.Model)

	err = myMod.Start(ctx)
	defer myMod.Close(ctx)
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}
