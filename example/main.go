package main

import (
	"github.com/china8036/gdht"
	"math/rand"
	"time"
	"log"
	"github.com/china8036/golog"
)

var cacheFile = "cache.log"

func main() {
	//fbyte, err := ioutil.ReadFile(cacheFile)
	//tmp_node := string(fbyte)
	//if !gdht.IsNodeId(tmp_node) {
	//	tmp_node = string(randNodeId())
	//	ioutil.WriteFile(cacheFile, []byte(tmp_node), os.ModePerm)
	//}
	ew := &golog.EvetWriter{}
	log.SetOutput(ew)
	dht, err := gdht.New()
	if err != nil {
		log.Println(err)
		return
	}
	dht.Run()
	time.Sleep(time.Second * 10)
	dht.GetPeers("91deb3de3c09a300e2b987efc4fa8508bbd924dc")
	dht.Wait()

}

func randNodeId() []byte {
	b := make([]byte, 20)
	rand.Read(b)
	return b
}
