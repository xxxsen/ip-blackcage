package ipset

type SetType string
type OutputType string

const (
	SetTypeHashNet SetType = "hash:net"
)

const (
	OutputTypePlain OutputType = "plain"
	OutputTypeSave  OutputType = "save"
	OutputTypeXml   OutputType = "xml"
)
