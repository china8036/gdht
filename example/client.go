package main

import ()
import (
	"encoding/hex"
	"github.com/china8036/gdht"
	"local/oauth2/lib"
	"fmt"
)

func main() {
	krpc, err := gdht.NewKrpc(lib.GenRandomString(20))
	if err !=nil{
		fmt.Println(err)
	}
	krpc.FindNode(krpc.NodeId,"127.0.0.1:6881",nil)
	krpc.Wait()

}


// DecodeInfoHash transforms a hex-encoded 20-characters string to a binary
// infohash.
func DecodeInfoHash(x string) (b string, err error) {//20位hash有时候是不可见字符
	var h []byte
	h, err = hex.DecodeString(x)
	return string(h), err
}

// DecodeInfoHash transforms a hex-encoded 20-characters string to a binary
// infohash.
func EncodeInfoHash(x string) string {//20位hash有时候是不可见字符 encode后转换为可见字符
	var h []byte
	hex.Encode(h,[]byte(x))
	return string(h)
}