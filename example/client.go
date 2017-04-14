package main

import ()
import (
	"fmt"
	"encoding/hex"
)

func main() {
	x := "deca7a89a1dbdc4b213de1c0d5351e92582f31fb"
	fmt.Println(DecodeInfoHash(x))

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