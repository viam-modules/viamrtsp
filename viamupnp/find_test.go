package viamupnp

import (
	"context"
	"testing"

	"github.com/koron/go-ssdp"
	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

var exampleXML = `<?xml version="1.0"?>
<root xmlns="urn:schemas-upnp-org:device-1-0">
  <specVersion>
    <major>1</major>
    <minor>0</minor>
  </specVersion>
  <device>
    <deviceType>urn:schemas-upnp-org:device:basic:1</deviceType>
    <friendlyName>M364C TAVY16D</friendlyName>
    <manufacturer>FLIR Systems, Inc.</manufacturer>
    <manufacturerURL>http://www.flir.com</manufacturerURL>
    <modelDescription>FLIR Infrared Camera</modelDescription>
    <modelName>M364C</modelName>
    <modelNumber>E70518</modelNumber>
    <modelURL>http://www.flir.com/marine/</modelURL>
    <serialNumber>TAVY16D</serialNumber>
    <UDN>uuid:IRCamera-1_0-89911a22-ef16-11dd-84a7-0011c7156eab</UDN>
    <UPC></UPC>
    <serviceList>
    </serviceList>
   <presentationURL>http://172.16.5.12:80/</presentationURL>
   <buildDate>00/00/00</buildDate>
   <deviceControl>JCU</deviceControl>
   <submodelName></submodelName>
   <deviceInfo>
     <CGIport>8090</CGIport>
     <ONVIFport>8091</ONVIFport>
     <productId></productId>
   </deviceInfo>
</device>
</root>`

func TestParse1(t *testing.T) {
	dd, err := parseDeviceDesc("", []byte(exampleXML))
	test.That(t, err, test.ShouldBeNil)
	test.That(t, dd.SpecVersion.Major, test.ShouldEqual, 1)
	test.That(t, dd.Device.ModelName, test.ShouldEqual, "M364C")
	test.That(t, dd.Device.Manufacturer, test.ShouldEqual, "FLIR Systems, Inc.")
	test.That(t, dd.Device.SerialNumber, test.ShouldEqual, "TAVY16D")
}

func TestMatches1(t *testing.T) {
	test.That(t, matches("a", "a"), test.ShouldBeTrue)
	test.That(t, matches("a", "abc"), test.ShouldBeFalse)
	test.That(t, matches("a.*", "abc"), test.ShouldBeTrue)
}

func TestQuery1(t *testing.T) {
	pc := UPNPDevice{
		Desc: &deviceDesc{},
	}

	pc.Desc.Device.Manufacturer = "ax"
	pc.Desc.Device.ModelName = "by"
	pc.Desc.Device.SerialNumber = "cz"

	test.That(t, pc.Matches(DeviceQuery{}), test.ShouldBeTrue)

	test.That(t, pc.Matches(DeviceQuery{Manufacturer: "ax"}), test.ShouldBeTrue)
	test.That(t, pc.Matches(DeviceQuery{Manufacturer: "a.*"}), test.ShouldBeTrue)
	test.That(t, pc.Matches(DeviceQuery{Manufacturer: "ax.*"}), test.ShouldBeTrue)
	test.That(t, pc.Matches(DeviceQuery{Manufacturer: "b"}), test.ShouldBeFalse)

	test.That(t, pc.Matches(DeviceQuery{ModelName: "by"}), test.ShouldBeTrue)
	test.That(t, pc.Matches(DeviceQuery{ModelName: "b.*"}), test.ShouldBeTrue)
	test.That(t, pc.Matches(DeviceQuery{ModelName: "by.*"}), test.ShouldBeTrue)
	test.That(t, pc.Matches(DeviceQuery{ModelName: "d"}), test.ShouldBeFalse)

	test.That(t, pc.Matches(DeviceQuery{SerialNumber: "cz"}), test.ShouldBeTrue)
	test.That(t, pc.Matches(DeviceQuery{SerialNumber: "c.*"}), test.ShouldBeTrue)
	test.That(t, pc.Matches(DeviceQuery{SerialNumber: "cz.*"}), test.ShouldBeTrue)
	test.That(t, pc.Matches(DeviceQuery{SerialNumber: "d"}), test.ShouldBeFalse)
}

func TestFindHost(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	ctx = context.WithValue(ctx,
		FindAllTestKey,
		[]UPNPDevice{
			{ssdp.Service{Location: "http://eliot:12312/asd.xml"}, nil},
		},
	)

	host, err := FindHost(ctx, logger, DeviceQuery{})
	test.That(t, err, test.ShouldBeNil)
	test.That(t, host, test.ShouldEqual, "eliot")
}
