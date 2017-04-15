package main

import (
	"fmt"
	"github.com/china8036/gdht"
	"math/rand"
	"time"
)

func main() {

	nodeid := string(randNodeId())
	dht,err := gdht.New(nodeid)
	if err!=nil{
		fmt.Println(err)
		return
	}
	dht.Run()
	time.Sleep(time.Second * 10)
	dht.GetPeers("deca7a89a1dbdc4b213de1c0d5351e92582f31fb")
	dht.Wait()

}

func randNodeId() []byte {
	b := make([]byte, 20)
	rand.Read(b)
	return b
}