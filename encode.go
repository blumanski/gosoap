package gosoap

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"regexp"
)

var (
	soapPrefix                            = "soap"
	customEnvelopeAttrs map[string]string = nil
)

// SetCustomEnvelope define customizated envelopefdsfds
func SetCustomEnvelope(prefix string, attrs map[string]string) {
	soapPrefix = prefix
	if attrs != nil {
		customEnvelopeAttrs = attrs
	}
}

// MarshalXML envelope the body and encode to xml dfdsfds
func (c process) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	tokens := &tokenData{}

	//start envelope
	if c.Client.Definitions == nil {
		return fmt.Errorf("definitions is nil")
	}

	namespace := ""
	if c.Client.Definitions.Types != nil {
		schema := c.Client.Definitions.Types[0].XsdSchema[0]
		namespace = c.Client.Definitions.TargetNamespace
		if namespace == "" && len(schema.Imports) > 0 {
			namespace = schema.Imports[0].Namespace
		}
	}

	tokens.startEnvelope()
	if len(c.Client.HeaderParams) > 0 {
		tokens.startHeader(c.Client.HeaderName, namespace)
		tokens.recursiveEncode(c.Client.HeaderParams)
		tokens.endHeader(c.Client.HeaderName)
	}

	err := tokens.startBody(c.Request.Method, namespace)
	if err != nil {
		return err
	}

	tokens.recursiveEncode(c.Request.Params)

	//end envelope
	tokens.endBody(c.Request.Method)
	tokens.endEnvelope()

	for _, t := range tokens.data {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}

	return e.Flush()
}

type tokenData struct {
	data []xml.Token
}

func (tokens *tokenData) recursiveEncode(hm interface{}) {
	v := reflect.ValueOf(hm)

	switch v.Kind() {
	case reflect.Map:

		regexpAttrBox := regexp.MustCompile(`([^\[]+)\[(.*)\]`)
		regexpAttrs := regexp.MustCompile(`,?([^=]+)=([^,]+)`)

		for _, key := range v.MapKeys() {
			if regexpAttrBox.MatchString(key.String()) {
				rs := regexpAttrBox.FindStringSubmatch(key.String())

				var attrs []xml.Attr
				matches := regexpAttrs.FindAllStringSubmatch(rs[2], -1)
				for _, match := range matches {
					attrs = append(attrs, xml.Attr{Name: xml.Name{Local: match[1]}, Value: match[2]})
				}

				t := xml.StartElement{
					Name: xml.Name{
						Space: "",
						Local: rs[1],
					},
					Attr: attrs,
				}
				tokens.data = append(tokens.data, t)
				tokens.recursiveEncode(v.MapIndex(key).Interface())
				tokens.data = append(tokens.data, xml.EndElement{Name: t.Name})
			} else {
				t := xml.StartElement{
					Name: xml.Name{
						Space: "",
						Local: key.String(),
					},
				}
				tokens.data = append(tokens.data, t)
				tokens.recursiveEncode(v.MapIndex(key).Interface())
				tokens.data = append(tokens.data, xml.EndElement{Name: t.Name})
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			tokens.recursiveEncode(v.Index(i).Interface())
		}
	case reflect.String:
		content := xml.CharData(v.String())
		tokens.data = append(tokens.data, content)
	}
}

func (tokens *tokenData) startEnvelope() {
	e := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Envelope", soapPrefix),
		},
	}

	if customEnvelopeAttrs == nil {
		e.Attr = []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns:xsi"}, Value: "http://www.w3.org/2001/XMLSchema-instance"},
			{Name: xml.Name{Space: "", Local: "xmlns:xsd"}, Value: "http://www.w3.org/2001/XMLSchema"},
			{Name: xml.Name{Space: "", Local: "xmlns:soap"}, Value: "http://schemas.xmlsoap.org/soap/envelope/"},
		}
	} else {
		e.Attr = make([]xml.Attr, 0)
		for local, value := range customEnvelopeAttrs {
			e.Attr = append(e.Attr, xml.Attr{
				Name:  xml.Name{Space: "", Local: local},
				Value: value,
			})
		}
	}

	tokens.data = append(tokens.data, e)
}

func (tokens *tokenData) endEnvelope() {
	e := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Envelope", soapPrefix),
		},
	}

	tokens.data = append(tokens.data, e)
}

func (tokens *tokenData) startHeader(m, n string) {
	h := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Header", soapPrefix),
		},
	}

	if m == "" || n == "" {
		tokens.data = append(tokens.data, h)
		return
	}

	r := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns"}, Value: n},
		},
	}

	tokens.data = append(tokens.data, h, r)
}

func (tokens *tokenData) endHeader(m string) {
	h := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Header", soapPrefix),
		},
	}

	if m == "" {
		tokens.data = append(tokens.data, h)
		return
	}

	r := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
	}

	tokens.data = append(tokens.data, r, h)
}

func (tokens *tokenData) startBody(m, n string) error {
	b := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Body", soapPrefix),
		},
	}

	if m == "" || n == "" {
		return fmt.Errorf("method or namespace is empty")
	}

	r := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns"}, Value: n},
		},
	}

	tokens.data = append(tokens.data, b, r)

	return nil
}

// endToken close body of the envelope
func (tokens *tokenData) endBody(m string) {
	b := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Body", soapPrefix),
		},
	}

