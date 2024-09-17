package viamrtsp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.viam.com/rdk/logging"
)

const (
	maxUDPMessageBytesSize     = 1536
	standardWSDiscoveryAddress = "239.255.255.250:3702"
	discoveryTimeout           = 10 * time.Second
)

// RTSPDiscovery is responsible for discovering RTSP camera devices using WS-Discovery and ONVIF.
type RTSPDiscovery struct {
	multicastAddress string
	logger           logging.Logger
	conn             *net.UDPConn
}

// newRTSPDiscovery creates a new RTSPDiscovery instance with default values.
func newRTSPDiscovery(logger logging.Logger) *RTSPDiscovery {
	return &RTSPDiscovery{
		multicastAddress: standardWSDiscoveryAddress,
		logger:           logger,
	}
}

// Close closes the UDP connection if it exists.
func (d *RTSPDiscovery) close() error {
	if d.conn != nil {
		err := d.conn.Close()
		d.conn = nil
		return err
	}
	return nil
}

// generateDiscoveryMessage formats an xml discovery message properly.
func (d *RTSPDiscovery) generateDiscoveryMessage() string {
	messageID := uuid.New().String()
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
	<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope" 
                xmlns:wsa="http://schemas.xmlsoap.org/ws/2004/08/addressing" 
                xmlns:wsdd="http://schemas.xmlsoap.org/ws/2005/04/discovery">
		<SOAP-ENV:Header>
			<wsa:MessageID>uuid:%s</wsa:MessageID>
			<wsa:To>urn:schemas-xmlsoap-org:ws:2005:04:discovery</wsa:To>
			<wsa:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</wsa:Action>
		</SOAP-ENV:Header>
		<SOAP-ENV:Body>
			<wsdd:Probe>
				<wsdd:Types>dn:NetworkVideoTransmitter</wsdd:Types>
			</wsdd:Probe>
		</SOAP-ENV:Body>
	</SOAP-ENV:Envelope>`, messageID)
}

// discoverRTSPAddresses performs a WS-Discovery and extracts RTSP addresses from the XAddrs field.
func (d *RTSPDiscovery) discoverRTSPAddresses() ([]string, error) {
	var discoveredAddresses []string

	addr, err := net.ResolveUDPAddr("udp4", d.multicastAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	d.conn, err = net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP socket: %w", err)
	}
	defer d.close()

	_, err = d.conn.WriteToUDP([]byte(d.generateDiscoveryMessage()), addr)
	if err != nil {
		return nil, fmt.Errorf("failed to send discovery message: %w", err)
	}

	if err = d.conn.SetReadDeadline(time.Now().Add(discoveryTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	buffer := make([]byte, maxUDPMessageBytesSize)
	for {
		n, _, err := d.conn.ReadFromUDP(buffer)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				d.logger.Debug("Discovery timed out after waiting.")
				return discoveredAddresses, nil
			}
			return nil, fmt.Errorf("error reading from UDP: %w", err)
		}

		response := buffer[:n]
		xaddrs := d.extractXAddrsFromProbeMatch(response)

		discoveredAddresses = append(discoveredAddresses, xaddrs...)
	}
}

// extractXAddrsFromProbeMatch extracts XAddrs from the WS-Discovery ProbeMatch response.
func (d *RTSPDiscovery) extractXAddrsFromProbeMatch(response []byte) []string {
	type ProbeMatch struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			ProbeMatches struct {
				ProbeMatch []struct {
					XAddrs string `xml:"XAddrs"`
				} `xml:"ProbeMatch"`
			} `xml:"ProbeMatches"`
		} `xml:"Body"`
	}

	var probeMatch ProbeMatch
	err := xml.NewDecoder(bytes.NewReader(response)).Decode(&probeMatch)
	if err != nil {
		d.logger.Warnf("error unmarshalling ONVIF discovery xml response: %w", err)
	}

	var xaddrs []string
	for _, match := range probeMatch.Body.ProbeMatches.ProbeMatch {
		for _, xaddr := range strings.Split(match.XAddrs, " ") {
			if strings.HasPrefix(xaddr, "http://") {
				xaddrs = append(xaddrs, xaddr)
			}
		}
	}

	return xaddrs
}
