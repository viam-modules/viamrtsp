// This package provides the entrypoint for the remote
package main

import (
	"context"
	"os"

	"github.com/erh/viamrtsp"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/config"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	robotimpl "go.viam.com/rdk/robot/impl"
	"go.viam.com/rdk/robot/web"
	rdkutils "go.viam.com/rdk/utils"
	"go.viam.com/utils"
)

func main() {
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("client"))
}

func mainWithArgs(ctx context.Context, _ []string, logger logging.Logger) error {
	netconfig := config.NetworkConfig{}
	netconfig.BindAddress = "0.0.0.0:8083"

	if err := netconfig.Validate(""); err != nil {
		return err
	}

	conf := &config.Config{
		Network: netconfig,
		Components: []resource.Config{
			{
				Name:  os.Args[1],
				API:   camera.API,
				Model: viamrtsp.ModelAgnostic,
				Attributes: rdkutils.AttributeMap{
					"rtsp_address": os.Args[2],
				},
				ConvertedAttributes: &viamrtsp.Config{
					Address: os.Args[2],
				},
			},
		},
	}

	myRobot, err := robotimpl.New(ctx, conf, logger)
	if err != nil {
		return err
	}

	return web.RunWebWithConfig(ctx, myRobot, conf, logger)
}
