package main

import (
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
	k.Ping("67.215.246.10:6881")
	k.Wait()

}