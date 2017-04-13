package main

import ()
import (
	"fmt"
)

func main() {
	var b [20]byte
	for i, _ := range b {
		b[i] = 0x00
	}
	test := b[0:]
	btest := append(test[:0],test[1:]...)
	fmt.Println(btest)
	fmt.Println(b[0:1])

}
