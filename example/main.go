package main

import (
	"fmt"
	"github.com/china8036/gdht"
	"math/rand"
)

func main() {

	nodeid := string(randNodeId())
	dht,err := gdht.New(nodeid)
	if err!=nil{
		fmt.Println(err)
		return
	}
	dht.Run()
	dht.Wait()

}

func randNodeId() []byte {
	b := make([]byte, 20)
	rand.Read(b)
	return b
}