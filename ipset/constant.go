package ipset

type SetType string
type OutputType string

const (
	HashNetType SetType = "hash:net"
)

const (
	OutTypePlain OutputType = "plain"
	OutTypeSave  OutputType = "save"
	OutTypeXml   OutputType = "xml"
)
