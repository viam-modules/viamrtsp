// This package is a binary for trying out onvif discovery
package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/koron/go-ssdp"
)

func main() {
	err := realMain()
	if err != nil {
		panic(err)
	}
}

// UPNPDevice is a UPNPDevice device.
type UPNPDevice struct {
	Service ssdp.Service
	Desc    *DeviceDesc
}

func realMain() error {
	network := "239.255.255.250:1900"
	timeout := 3
	mode := "search"
	var debug bool

	flag.StringVar(&network, "network", network, "network to cast to")
	flag.IntVar(&timeout, "timeout", timeout, "number of seconds to listen to upnp search")
	flag.StringVar(&mode, "mode", mode, "the mode you want to run, search | describe")
	flag.BoolVar(&debug, "debug", debug, "enable debug mode")

	flag.Parse()
	services, err := ssdp.Search(ssdp.All, timeout, network)
	if err != nil {
		return err
	}
	if mode == "search" {
		j, err := json.Marshal(services)
		if err != nil {
			return err
		}
		log.Print(string(j))
		return nil
	}

	devices := map[string]{[]UPNPDevice{}}
	for _, s := range services {
		log.Printf("%s request start\n", s.Location)
		resp, err := readDeviceDesc(s.Location, time.Second*10)
		if err != nil {
			log.Printf("%s request fail err: %s\n", s.Location, err.Error())
			devices = append(devices, UPNPDevice{Service: s})
			continue
		}
		log.Printf("response from %s\n", s.Location)
		log.Println(string(resp))

		var desc DeviceDesc
		if err := xml.Unmarshal(resp, &desc); err != nil {
			log.Printf("failed to parse xml response from %s, err: %s\n", s.Location, err.Error())
			devices = append(devices, UPNPDevice{Service: s})
			continue
		}
		devices = append(devices, UPNPDevice{Service: s, Desc: &desc})
	}
	j, err := json.Marshal(devices)
	if err != nil {
		return err
	}
	fmt.Print(string(j))
	return nil
}

type DeviceDesc struct {
	XMLName     xml.Name `xml:"root"  json:"root"`
	SpecVersion struct {
		Major int `xml:"major" json:"major"`
		Minor int `xml:"minor" json:"minor"`
	} `xml:"specVersion" json:"specVersion"`
	Device struct {
		Manufacturer string `xml:"manufacturer" json:"manufacturer"`
		ModelName    string `xml:"modelName" json:"modelName"`
		SerialNumber string `xml:"serialNumber" json:"serialNumber"`
	} `xml:"device" json:"device"`
}

func readDeviceDesc(url string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cli := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't fetch xml(%s): %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http fetch (%s) not ok: %v", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read body from (%s): %v", url, resp.StatusCode)
	}

	return data, nil
}

func parseDeviceDesc(data []byte) (*DeviceDesc, error) {
	var desc DeviceDesc
	err := xml.Unmarshal(data, &desc)
	if err != nil {
		return nil, err
	}

	return &desc, nil
}
