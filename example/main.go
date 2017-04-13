package main

import (
	"net"
	"fmt"
	"github.com/china8036/gdht"
	"local/oauth2/lib"
)

func main() {
	nodeid := lib.GenRandomString(20)
	k,err := gdht.New(nodeid)
	if err!=nil{
		fmt.Println(err)
		return
	}
	//DHTRouters:              "router.magnets.im:6881,router.bittorrent.com:6881,dht.transmissionbt.com:6881",
	ip := net.ParseIP("67.215.246.10")
	remote := net.UDPAddr{IP: ip, Port: 6881}
	q := gdht.QueryMessage{T: lib.GenRandomString(3), Y: "q", Q: "ping", A: map[string]interface{}{"id": nodeid}}
	k.SendMsg(remote, q)
	k.SendMsg(remote, q)
	k.SendMsg(remote, q)
	k.Wait()

}
