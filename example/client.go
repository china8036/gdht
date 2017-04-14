package main

import ()
import (
	"fmt"
	"encoding/binary"
	"bytes"
)

func main() {
	var b [2]byte
	b[0]=35
	b[1] = 173
	fmt.Println(b)
	bytesBuffer := bytes.NewBuffer(b[0:])
	var tmp int16
	binary.Read(bytesBuffer, binary.BigEndian, &tmp)
	fmt.Println(tmp)
	fmt.Println(int(tmp))

}
