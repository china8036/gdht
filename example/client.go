package main

import ()
import (
	"encoding/hex"
	"github.com/china8036/gdht"
	"local/oauth2/lib"
	"net"
	"fmt"
	"strings"
	"strconv"
	"encoding/binary"
	"bytes"
)

func main() {

	krpc, err := gdht.NewKrpc(lib.GenRandomString(20))
	if err != nil {
		fmt.Println(err)
	}
	krpc.FindNode(krpc.NodeId, "127.0.0.1:6881", nil)
	krpc.Wait()

}

//node结构
type node struct {
	addr   *net.UDPAddr
	nodeid string
}

func Test() {
	laddr, _ := net.ResolveUDPAddr(gdht.P, "127.0.0.1:80")
	en := node{laddr, lib.GenRandomString(20)}
	ips := en.addr.IP.String()
	ipslice := strings.Split(ips, ".")
	var ipbyte [4]byte
	for i, es := range ipslice {
		esi, _ := strconv.Atoi(es)
		ipbyte[i] = byte(esi)
	}

	tmp := int16(en.addr.Port)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, tmp)
	all := append([]byte(en.nodeid),ipbyte[0:]...)
	all = append(all,bytesBuffer.Bytes()...)
	fmt.Println(all)
	return
}

// DecodeInfoHash transforms a hex-encoded 20-characters string to a binary
// infohash.
func DecodeInfoHash(x string) (b string, err error) { //20位hash有时候是不可见字符
	var h []byte
	h, err = hex.DecodeString(x)
	return string(h), err
}

// DecodeInfoHash transforms a hex-encoded 20-characters string to a binary
// infohash.
func EncodeInfoHash(x string) string { //20位hash有时候是不可见字符 encode后转换为可见字符
	var h []byte
	hex.Encode(h, []byte(x))
	return string(h)
}
