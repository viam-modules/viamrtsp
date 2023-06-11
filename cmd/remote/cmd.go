package main

import (
	"context"
	"os"

	"github.com/edaniels/golog"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/rtsp"
	"go.viam.com/rdk/config"
	"go.viam.com/rdk/resource"
	robotimpl "go.viam.com/rdk/robot/impl"
	"go.viam.com/rdk/robot/web"
	"go.viam.com/rdk/utils"

	"github.com/erh/viamrtsp"
)

func main() {
	err := realMain()
	if err != nil {
		panic(err)
	}
}
func realMain() error {

	ctx := context.Background()
	logger := golog.NewDevelopmentLogger("client")

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
				Model: viamrtsp.ModelH264,
				Attributes: utils.AttributeMap{
					"rtsp_address": os.Args[2],
				},
				ConvertedAttributes: &rtsp.Config{
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
