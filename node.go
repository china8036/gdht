package gdht

import (
	"net"
	"fmt"
)


const (
	NodeIdLen = 20
)

//node结构
type node struct {
	addr *net.UDPAddr
	nodeid   string
}

//判断是否是nodeid
func IsNodeId(nodeid string) bool{
	return len(nodeid) == NodeIdLen
}

// Calculates the distance between two hashes. In DHT/Kademlia, "distance" is
// the XOR of the torrent infohash and the peer node ID.  This is slower than
// necessary. Should only be used for displaying friendly messages.
func NodeDistance(id1 string, id2 string) (distance string) {
	d := make([]byte, len(id1))
	fmt.Printf("id1 %x\n",id1)
	fmt.Printf("id2 %x\n",id2)
	if len(id1) != len(id2) {
		return ""
	} else {
		for i := 0; i < len(id1); i++ {
			d[i] = id1[i] ^ id2[i]
		}
		fmt.Printf("dis %x \n",string(d))
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

