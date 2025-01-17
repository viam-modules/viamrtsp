// Package gosoap provides a minimal soap client for viamonvif
package gosoap

import (
	//nolint: gosec
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"time"

	// TODO: Remove dependence on etree.
	"github.com/beevik/etree"
	"github.com/elgs/gostrgen"
)

// SoapMessage type from string.
type SoapMessage string

// NewEmptySOAP return new SoapMessage.
func NewEmptySOAP() (SoapMessage, error) {
	var zero SoapMessage
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	env := doc.CreateElement("soap-env:Envelope")
	env.CreateElement("soap-env:Header")
	env.CreateElement("soap-env:Body")
	env.CreateAttr("xmlns:soap-env", "http://www.w3.org/2003/05/soap-envelope")
	env.CreateAttr("xmlns:soap-enc", "http://www.w3.org/2003/05/soap-encoding")
	res, err := doc.WriteToString()
	if err != nil {
		return zero, err
	}

	return SoapMessage(res), nil
}

func (msg SoapMessage) String() string {
	return string(msg)
}

// AddBodyContent for Envelope.
func (msg *SoapMessage) AddBodyContent(element *etree.Element) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(msg.String()); err != nil {
		return err
	}
	bodyTag := doc.Root().SelectElement("Body")
	bodyTag.AddChild(element)

	res, err := doc.WriteToString()
	if err != nil {
		return err
	}

	*msg = SoapMessage(res)
	return nil
}

// AddStringHeaderContent for Envelope body.
func (msg *SoapMessage) AddStringHeaderContent(data string) error {
	doc := etree.NewDocument()

	if err := doc.ReadFromString(data); err != nil {
		return err
	}

	element := doc.Root()
	doc = etree.NewDocument()
	if err := doc.ReadFromString(msg.String()); err != nil {
		return err
	}

	bodyTag := doc.Root().SelectElement("Header")
	bodyTag.AddChild(element)

	res, err := doc.WriteToString()
	if err != nil {
		return err
	}
	*msg = SoapMessage(res)

	return nil
}

// AddRootNamespace for Envelope body.
func (msg *SoapMessage) AddRootNamespace(key, value string) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(msg.String()); err != nil {
		return err
	}
	doc.Root().CreateAttr("xmlns:"+key, value)
	res, err := doc.WriteToString()
	if err != nil {
		return err
	}

	*msg = SoapMessage(res)
	return nil
}

// AddWSSecurity Header for soapMessage.
func (msg *SoapMessage) AddWSSecurity(username, password string) error {
	//doc := etree.NewDocument()
	//if err := doc.ReadFromString(msg.String()); err != nil {
	//	log.Println(err.Error())
	//}
	/*
		Getting an WS-Security struct representation
	*/
	auth := newSecurity(username, password)

	/*
		Adding WS-Security namespaces to root element of SOAP message
	*/
	//msg.AddRootNamespace("wsse", "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext1.0.xsd")
	//msg.AddRootNamespace("wsu", "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility1.0.xsd")

	soapReq, err := xml.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}

	/*
		Adding WS-Security struct to SOAP header
	*/
	if err := msg.AddStringHeaderContent(string(soapReq)); err != nil {
		return err
	}
	return nil
}

// AddAction Header handling for soapMessage.
func (msg *SoapMessage) AddAction() error {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(msg.String()); err != nil {
		return err
	}
	return nil
}

/*
************************

	WS-Security types

************************.
*/
const (
	//nolint: gosec
	passwordType = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest"
	// nolint: gosec
	encodingType = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary"
)

// Security type :XMLName xml.Name `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`.
type Security struct {
	// XMLName xml.Name  `xml:"wsse:Security"`
	XMLName xml.Name `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd Security"`
	Auth    wsAuth
}

type password struct {
	// XMLName xml.Name `xml:"wsse:Password"`
	Type     string `xml:"Type,attr"`
	Password string `xml:",chardata"`
}

type nonce struct {
	// XMLName xml.Name `xml:"wsse:Nonce"`
	Type  string `xml:"EncodingType,attr"`
	Nonce string `xml:",chardata"`
}

type wsAuth struct {
	XMLName  xml.Name `xml:"UsernameToken"`
	Username string   `xml:"Username"`
	Password password `xml:"Password"`
	Nonce    nonce    `xml:"Nonce"`
	Created  string   `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd Created"`
}

//nolint: lll
/*
   <Security s:mustUnderstand="1" xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
       <UsernameToken>
           <Username>admin</Username>
           <Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">edBuG+qVavQKLoWuGWQdPab4IBE=</Password>
           <Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">S7wO1ZFTh0KXv2CR7bd2ZXkLAAAAAA==</Nonce>
           <Created xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">2018-04-10T18:04:25.836Z</Created>
       </UsernameToken>
   </Security>
*/

// newSecurity get a new security.
func newSecurity(username, passwd string) Security {
	/** Generating Nonce sequence **/
	charsToGenerate := 32
	charSet := gostrgen.Lower | gostrgen.Digit

	nonceSeq, _ := gostrgen.RandGen(charsToGenerate, charSet, "", "")
	created := time.Now().UTC().Format(time.RFC3339Nano)
	auth := Security{
		Auth: wsAuth{
			Username: username,
			Password: password{
				Type:     passwordType,
				Password: generateToken(nonceSeq, created, passwd),
			},
			Nonce: nonce{
				Type:  encodingType,
				Nonce: nonceSeq,
			},
			Created: created,
		},
	}

	return auth
}

// Digest = B64ENCODE( SHA1( B64DECODE( Nonce ) + Date + Password ) ).
func generateToken(Nonce string, Created string, Password string) string {
	sDec, _ := base64.StdEncoding.DecodeString(Nonce)

	//nolint: gosec
	hasher := sha1.New()
	hasher.Write([]byte(string(sDec) + Created + Password))

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}
