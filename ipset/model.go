package ipset

import "encoding/xml"

type Ipsets struct {
	XMLName xml.Name `xml:"ipsets"`
	Ipset   []Ipset  `xml:"ipset"`
}

type Ipset struct {
	XMLName  xml.Name `xml:"ipset"`
	Name     string   `xml:"name,attr"`
	Type     string   `xml:"type"`
	Revision int      `xml:"revision"`
	Header   Header   `xml:"header"`
	Members  Members  `xml:"members"`
}

type Header struct {
	Family     string `xml:"family"`
	Hashsize   int    `xml:"hashsize"`
	Maxelem    int    `xml:"maxelem"`
	Memsize    int    `xml:"memsize"`
	References int    `xml:"references"`
	Numentries int    `xml:"numentries"`
}

type Members struct {
	Member []Member `xml:"member"`
}

type Member struct {
	Elem string `xml:"elem"`
}
