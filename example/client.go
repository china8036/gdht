package main

import ()
import (
	"fmt"
	"math"
)

func main() {
	var b [20]byte
	for i, _ := range b {
		b[i] = 0x00
	}
	m := 6
	mi := int(m % 8)
	index := 19 - int(m/8) //
	b[index] = byte(int(math.Pow(2, float64(mi))))
	fmt.Printf("%x\n", b)
	fmt.Printf("%s\n", string(b[0:]))

}
