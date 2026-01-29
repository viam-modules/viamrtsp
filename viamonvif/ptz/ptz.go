// Package ptz provides PTZ (Pan-Tilt-Zoom) ONVIF request and response types.
package ptz

import (
	"github.com/viam-modules/viamrtsp/viamonvif/xsd"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
)

// --- PTZ Request Types ---

// Stop is a request to stop PTZ movement.
type Stop struct {
	XMLName      string               `xml:"tptz:Stop"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	PanTilt      xsd.Boolean          `xml:"tptz:PanTilt,omitempty"`
	Zoom         xsd.Boolean          `xml:"tptz:Zoom,omitempty"`
}

// ContinuousMove is a request for continuous PTZ movement.
type ContinuousMove struct {
	XMLName      string               `xml:"tptz:ContinuousMove"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	Velocity     onvif.PTZSpeed       `xml:"tptz:Velocity"`
	Timeout      xsd.Duration         `xml:"tptz:Timeout,omitempty"`
}

// RelativeMove is a request for relative PTZ movement.
type RelativeMove struct {
	XMLName      string               `xml:"tptz:RelativeMove"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	Translation  onvif.PTZVector      `xml:"tptz:Translation"`
	Speed        onvif.PTZSpeed       `xml:"tptz:Speed,omitempty"`
}

// AbsoluteMove is a request for absolute PTZ movement.
type AbsoluteMove struct {
	XMLName      string               `xml:"tptz:AbsoluteMove"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
	Position     onvif.PTZVector      `xml:"tptz:Position"`
	Speed        onvif.PTZSpeed       `xml:"tptz:Speed,omitempty"`
}

// GetStatus is a request to get PTZ status.
type GetStatus struct {
	XMLName      string               `xml:"tptz:GetStatus"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
}

// GetConfiguration is a request to get PTZ configuration for a profile.
type GetConfiguration struct {
	XMLName      string               `xml:"tptz:GetConfiguration"`
	ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
}

// GetConfigurations is a request to get all PTZ configurations.
type GetConfigurations struct {
	XMLName string `xml:"tptz:GetConfigurations"`
}

// GetServiceCapabilities is a request to get PTZ service capabilities.
type GetServiceCapabilities struct {
	XMLName string `xml:"tptz:GetServiceCapabilities"`
}

// GetNodes is a request to get PTZ nodes.
type GetNodes struct {
	XMLName string `xml:"tptz:GetNodes"`
}

// --- PTZ Response Types ---

// Capabilities represents PTZ service capabilities.
type Capabilities struct {
	EFlip                       xsd.Boolean `xml:"EFlip,attr"`
	Reverse                     xsd.Boolean `xml:"Reverse,attr"`
	GetCompatibleConfigurations xsd.Boolean `xml:"GetCompatibleConfigurations,attr"`
	MoveStatus                  xsd.Boolean `xml:"MoveStatus,attr"`
	StatusPosition              xsd.Boolean `xml:"StatusPosition,attr"`
}

// GetServiceCapabilitiesResponse is the response to GetServiceCapabilities.
type GetServiceCapabilitiesResponse struct {
	Capabilities Capabilities `xml:"Capabilities"`
}

// GetConfigurationResponse is the response to GetConfiguration.
type GetConfigurationResponse struct {
	PTZConfiguration onvif.PTZConfiguration `xml:"PTZConfiguration"`
}

// GetConfigurationsResponse is the response to GetConfigurations.
type GetConfigurationsResponse struct {
	PTZConfiguration []onvif.PTZConfiguration `xml:"PTZConfiguration"`
}
