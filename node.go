package gdht

import (
	"net"
	"math"
	"fmt"
)



//node结构
type node struct {
	addr net.UDPAddr
	id   string
}


// Calculates the distance between two hashes. In DHT/Kademlia, "distance" is
// the XOR of the torrent infohash and the peer node ID.  This is slower than
// necessary. Should only be used for displaying friendly messages.
func HashDistance(id1 string, id2 string) (distance string) {
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
func Min(d1, d2 string) string {
	if d1 < d2 {
		return d1
	}
	return d2
}

//找出这个距离的index值
func FindIndex(d1 string) int {
	var b [20]byte
	for i, _ := range b {
		b[i] = 0x00
	}
	for m := 0; m < MaxKbNum; m++ {
		fmt.Printf("%x\n%x\n%x\n\n",d1,FindIndexLimit(m),FindIndexLimit(m+1))
		if d1 >= FindIndexLimit(m) && d1 <= FindIndexLimit(m+1) {
			return m
		}
	}
	return MaxKbNum

}

func FindIndexLimit(i int) (s string) {
	i = i % 160 //确认不出界
	var b [20]byte
	for index,_:=range b{
		b[index] = 0x00
	}
	mi := int(i % 8)
	index := 19 - int(i/8) //
	b[index] = byte(int(math.Pow(2, float64(mi))))
	return string(b[0:])
}
