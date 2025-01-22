package viamonvif

// inspired by https://github.com/use-go/onvif

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"go.viam.com/rdk/logging"
	"golang.org/x/net/ipv4"
)

const bufSize = 8192

var template = `<?xml version="1.0" encoding="UTF-8"?>
<e:Envelope xmlns:e="http://www.w3.org/2003/05/soap-envelope"
            xmlns:w="http://schemas.xmlsoap.org/ws/2004/08/addressing"
            xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery"
            xmlns:dn="http://www.onvif.org/ver10/network/wsdl">
 <e:Header>
  <w:MessageID>uuid:%s</w:MessageID>
  <w:To e:mustUnderstand="true">urn:schemas-xmlsoap-org:ws:2005:04:discovery</w:To>
  <w:Action a:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</w:Action>
 </e:Header>
 <e:Body>
  <d:Probe>
   <d:Types>dn:NetworkVideoTransmitter</d:Types>
  </d:Probe>
 </e:Body>
</e:Envelope>`

var (
	port = 3702
	//nolint: mnd
	group        = net.IPv4(239, 255, 255, 250)
	multicastTTL = 2
)

// SendProbe makes a WS-Discovery Probe call.
func SendProbe(interfaceName string, logger logging.Logger) ([]string, error) {
	logger.Debug("Starting SendProbe")
	msg := fmt.Sprintf(template, uuid.Must(uuid.NewV4()).String())
	logger.Debug("listening on udp4 0.0.0.0:0")
	c, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}
	defer c.Close()

	logger.Debugf("looking up interface %s\n", interfaceName)
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	p := ipv4.NewPacketConn(c)

	if err := p.JoinGroup(iface, &net.UDPAddr{IP: group}); err != nil {
		return nil, err
	}

	dst := &net.UDPAddr{IP: group, Port: port}
	logger.Debugf("SendProbe sending message on interface %s, multicast: %s\n", interfaceName, dst.String())
	logger.Debug(msg)
	data := []byte(msg)
	if err := p.SetMulticastInterface(iface); err != nil {
		return nil, err
	}
	if err := p.SetMulticastTTL(multicastTTL); err != nil {
		return nil, err
	}
	if _, err := p.WriteTo(data, nil, dst); err != nil {
		return nil, err
	}

	//nolint:mnd
	if err := p.SetReadDeadline(time.Now().Add(time.Second * 2)); err != nil {
		return nil, err
	}

	var result []string
	for {
		b := make([]byte, bufSize)
		n, _, _, err := p.ReadFrom(b)
		if err != nil {
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				return nil, err
			}
			break
		}
		result = append(result, string(b[0:n]))
	}
	return result, nil
}
