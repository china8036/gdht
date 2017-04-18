package gdht

import (
	"net"
	"qiniupkg.com/x/errors.v7"
)

const (
	NodeIdLen = 20
)

//node结构
type node struct {
	Addr   *net.UDPAddr
	Nodeid string
}

//新的node
func NewNode(addr *net.UDPAddr, nodeid string) (*node, error) {

	if !IsNodeId(nodeid) {
		return nil, errors.New("ivalid node id")
	}
	return &node{addr, nodeid}, nil
}

//判断是否是nodeid
func IsNodeId(nodeid string) bool {
	return len(nodeid) == NodeIdLen
}

// Calculates the distance between two hashes. In DHT/Kademlia, "distance" is
// the XOR of the torrent infohash and the peer node ID.  This is slower than
// necessary. Should only be used for displaying friendly messages.
func NodeDistance(id1 string, id2 string) (distance string) {
	d := make([]byte, len(id1))
	if len(id1) != len(id2) {
		return ""
	} else {
		for i := 0; i < len(id1); i++ {
			d[i] = id1[i] ^ id2[i]
		}
		return string(d)
	}

	return ""
}

//只用来比较大小
func MinDistance(d1, d2 string) string {
	if d1 < d2 {
		return d1
	}
	return d2
}

//找出最近的几个nodes
func FindMinDistanceNodes(num int, nodeids []node, targetid string) []node {
	if len(nodeids) < num {
		return nodeids
	}
	returnnodes := make([]node, num)

	for i := 0; i < num; i++ {
		inode, index := FindMinDistanceNode(nodeids, targetid)
		returnnodes[i] = *inode
		nodeids = append(nodeids[0:index], nodeids[index+1:]...) //等于删除index

	}
	return returnnodes

}

//找到最小距离的nodeid
//node slice and index int
func FindMinDistanceNode(nodeids []node, targetid string) (*node, int) {
	if nodeids == nil {
		return nil, 0
	}
	min := nodeids[0]
	index := 0
	mindis := NodeDistance(min.Nodeid, targetid)
	for i, v := range nodeids {
		if i == 0 {
			continue //
		}
		tdis := NodeDistance(v.Nodeid, targetid)
		if tdis < mindis {
			min = v
			mindis = tdis
			index = i
		}
	}
	return &min, index
}
