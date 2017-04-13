package main

import (
	"fmt"
	"github.com/china8036/gdht"
	"local/oauth2/lib"
)

func main() {
	nodeid := lib.GenRandomString(20)
	dht,err := gdht.New(nodeid)
	if err!=nil{
		fmt.Println(err)
		return
	}
	dht.Run()//"router.magnets.im:6881,router.bittorrent.com:6881,dht.transmissionbt.com:6881",

}