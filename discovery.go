package viamrtsp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

const (
    multicastAddress = "239.255.255.250:3702" // Standard WS-Discovery multicast address
    discoveryMessage = `<?xml version="1.0" encoding="utf-8"?>
    <e:Envelope xmlns:e="http://www.w3.org/2003/05/soap-envelope" xmlns:w="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery">
        <e:Header>
            <w:MessageID>uuid:5f9ecf95-ff0b-4869-801f-f7e19fb96e7c</w:MessageID>
            <w:To>urn:schemas-xmlsoap-org:ws:2005:04:discovery</w:To>
            <w:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</w:Action>
        </e:Header>
        <e:Body>
            <d:Probe>
                <d:Types>dn:NetworkVideoTransmitter</d:Types>
            </d:Probe>
        </e:Body>
    </e:Envelope>`
)

// discoverRTSPAddresses performs a WS-Discovery and extracts RTSP addresses from the XAddrs field
func discoverRTSPAddresses() ([]string, error) {
    var discoveredAddresses []string

    addr, err := net.ResolveUDPAddr("udp4", multicastAddress)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve UDP address: %v", err)
    }

    conn, err := net.ListenUDP("udp4", nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create UDP socket: %v", err)
    }
    defer conn.Close()

    _, err = conn.WriteToUDP([]byte(discoveryMessage), addr)
    if err != nil {
        return nil, fmt.Errorf("failed to send discovery message: %v", err)
    }

    conn.SetReadDeadline(time.Now().Add(5 * time.Second))

    buffer := make([]byte, 8192)
    for {
        n, _, err := conn.ReadFromUDP(buffer)
        if err != nil {
            if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
                break
            }
            return nil, fmt.Errorf("error reading from UDP: %v", err)
        }

        response := buffer[:n]
        xaddrs, err := extractXAddrsFromProbeMatch(response)
        if err != nil {
            log.Printf("Failed to parse response: %v\n", err)
            continue
        }

        discoveredAddresses = append(discoveredAddresses, xaddrs...)
    }

    return discoveredAddresses, nil
}

// extractXAddrsFromProbeMatch extracts XAddrs from the WS-Discovery ProbeMatch response
func extractXAddrsFromProbeMatch(response []byte) ([]string, error) {
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
        return nil, fmt.Errorf("error unmarshalling probe match: %v", err)
    }

    var xaddrs []string
    for _, match := range probeMatch.Body.ProbeMatches.ProbeMatch {
        for _, xaddr := range strings.Split(match.XAddrs, " ") {
            if strings.HasPrefix(xaddr, "rtsp://") {
                xaddrs = append(xaddrs, xaddr)
            }
        }
    }

    return xaddrs, nil
}
