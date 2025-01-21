// This package is a binary for trying out onvif discovery
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/viam-modules/viamrtsp"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/utils"
)

type options struct {
	output string
	url    string
}

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err.Error())
	}
}

func realMain() error {
	logger := logging.NewDebugLogger("discovery")
	//nolint:mnd
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	opts := parseOpts()
	config := resource.NewEmptyConfig(camera.Named("foo"), viamrtsp.ModelAgnostic)
	config.ConvertedAttributes = &viamrtsp.Config{Address: opts.url}
	cam, err := viamrtsp.NewRTSPCamera(ctx, nil, config, logger)
	if err != nil {
		return err
	}
	defer cam.Close(ctx)

	var (
		b  []byte
		md camera.ImageMetadata
	)
	for {
		if ctx.Err() != nil {
			break
		}

		b, md, err = cam.Image(ctx, utils.MimeTypeJPEG, nil)
		if err != nil {
			continue
		}
		if md.MimeType != utils.MimeTypeJPEG {
			return fmt.Errorf("expected cam.Image to return mime_type: %s, instead got %s", utils.MimeTypeJPEG, md.MimeType)
		}
		break
	}

	if err != nil {
		return err
	}

	//nolint:mnd
	return os.WriteFile(opts.output, b, 0o600)
}

func parseOpts() options {
	output := "output.jpeg"
	uri := ""

	flag.StringVar(&output, "o", output, "output file")
	flag.StringVar(&uri, "i", uri, "uri")
	flag.Parse()

	if uri == "" {
		flag.PrintDefaults()
		//nolint:forbidigo
		fmt.Println("missing required flag -i")
		//nolint:mnd
		os.Exit(2)
	}

	return options{
		url:    uri,
		output: output,
	}
}
