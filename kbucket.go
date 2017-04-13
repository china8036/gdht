package gdht


const (
	EachKbMaxLen = 8
	MaxKbNum     = 160
)

//一个K桶
type kbucket struct {
	index int
	nodes [EachKbMaxLen]node
}

