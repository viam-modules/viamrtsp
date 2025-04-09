//go:build ignore

// unrelated to the ptz-client. just a quick manual test to see if the xml parser works
//
//	Position:
//	  Pan/Tilt: X=-0.006167, Y=-0.087444 (Space: http://www.onvif.org/ver10/tptz/PanTiltSpaces/PositionGenericSpace)
//	  Zoom:     X=0.000000 (Space: http://www.onvif.org/ver10/tptz/ZoomSpaces/PositionGenericSpace)
//	Move Status:
//	  Pan/Tilt: UNKNOWN
//	  Zoom:     UNKNOWN
//	UTC Time:   2024-10-21T20:10:30Z
package main

import (
	"encoding/xml"
	"fmt"
	"log"
)

type CustomGetStatusEnvelope struct {
	XMLName xml.Name            `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
	Body    CustomGetStatusBody `xml:"http://www.w3.org/2003/05/soap-envelope Body"`
}

type CustomGetStatusBody struct {
	GetResponse CustomGetStatusResponse `xml:"http://www.onvif.org/ver20/ptz/wsdl GetStatusResponse"`
}

type CustomGetStatusResponse struct {
	PTZStatus CustomPTZStatus `xml:"http://www.onvif.org/ver20/ptz/wsdl PTZStatus"`
}

type CustomPTZStatus struct {
	Position   CustomPosition   `xml:"http://www.onvif.org/ver10/schema Position"`
	MoveStatus CustomMoveStatus `xml:"http://www.onvif.org/ver10/schema MoveStatus"`
	UtcTime    string           `xml:"http://www.onvif.org/ver10/schema UtcTime"`
}

type CustomPosition struct {
	PanTilt CustomVector2D `xml:"http://www.onvif.org/ver10/schema PanTilt"`
	Zoom    CustomVector1D `xml:"http://www.onvif.org/ver10/schema Zoom"`
}

type CustomVector2D struct {
	X     float64 `xml:"x,attr"`
	Y     float64 `xml:"y,attr"`
	Space string  `xml:"space,attr,omitempty"`
}

type CustomVector1D struct {
	X     float64 `xml:"x,attr"`
	Space string  `xml:"space,attr,omitempty"`
}

type CustomMoveStatus struct {
	PanTilt string `xml:"http://www.onvif.org/ver10/schema PanTilt"`
	Zoom    string `xml:"http://www.onvif.org/ver10/schema Zoom"`
}

var sampleXML = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope" xmlns:tds="http://www.onvif.org/ver10/device/wsdl" xmlns:trt="http://www.onvif.org/ver10/media/wsdl" xmlns:tptz="http://www.onvif.org/ver20/ptz/wsdl" xmlns:tev="http://www.onvif.org/ver10/events/wsdl" xmlns:timg="http://www.onvif.org/ver20/imaging/wsdl" xmlns:tmd="http://www.onvif.org/ver10/deviceIO/wsdl" xmlns:tth="http://www.onvif.org/ver10/thermal/wsdl" xmlns:tan="http://www.onvif.org/ver20/analytics/wsdl" xmlns:tt="http://www.onvif.org/ver10/schema" xmlns:axt="http://www.onvif.org/ver20/analytics" xmlns:ttr="http://www.onvif.org/ver20/analytics" xmlns:wsnt="http://docs.oasis-open.org/wsn/b-2" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery" xmlns:ter="http://www.onvif.org/ver10/error" xmlns:wsa="http://www.w3.org/2005/08/addressing" xmlns:wstop="http://docs.oasis-open.org/wsn/t-1" xmlns:wsrf-bf="http://docs.oasis-open.org/wsrf/bf-2" xmlns:wsrf-r="http://docs.oasis-open.org/wsrf/r-2" xmlns:tns1="http://www.onvif.org/ver10/topics" xmlns:tnsioi="http://dvtel.com/ioimage/event/topics" xmlns:tflir="http://www.flir.com/security/thermal/wsdl" xmlns:tnsflir="http://flir.com/flir/security/event/topics">
  <soap:Body>
    <tptz:GetStatusResponse>
      <tptz:PTZStatus>
        <tt:Position>
          <tt:PanTilt x="-0.006167" y="-0.087444" space="http://www.onvif.org/ver10/tptz/PanTiltSpaces/PositionGenericSpace"/>
          <tt:Zoom x="0.000000" space="http://www.onvif.org/ver10/tptz/ZoomSpaces/PositionGenericSpace"/>
        </tt:Position>
        <tt:MoveStatus>
          <tt:PanTilt>UNKNOWN</tt:PanTilt>
          <tt:Zoom>UNKNOWN</tt:Zoom>
        </tt:MoveStatus>
        <tt:UtcTime>2024-10-21T20:10:30Z</tt:UtcTime>
      </tptz:PTZStatus>
    </tptz:GetStatusResponse>
  </soap:Body>
</soap:Envelope>`

func main() {
	var statusEnvelope CustomGetStatusEnvelope
	// Use xml.Unmarshal with the custom struct and sample data
	err := xml.Unmarshal([]byte(sampleXML), &statusEnvelope)
	if err != nil {
		log.Fatalf("Failed to unmarshal sample GetStatus XML: %v", err)
	}

	ptzStatus := statusEnvelope.Body.GetResponse.PTZStatus

	fmt.Printf("  Position:\n")
	fmt.Printf("    Pan/Tilt: X=%.6f, Y=%.6f (Space: %s)\n",
		ptzStatus.Position.PanTilt.X,
		ptzStatus.Position.PanTilt.Y,
		ptzStatus.Position.PanTilt.Space)
	fmt.Printf("    Zoom:     X=%.6f (Space: %s)\n",
		ptzStatus.Position.Zoom.X,
		ptzStatus.Position.Zoom.Space)
	fmt.Printf("  Move Status:\n")
	fmt.Printf("    Pan/Tilt: %s\n", ptzStatus.MoveStatus.PanTilt)
	fmt.Printf("    Zoom:     %s\n", ptzStatus.MoveStatus.Zoom)
	fmt.Printf("  UTC Time:   %s\n", ptzStatus.UtcTime)
}
