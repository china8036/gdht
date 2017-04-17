package main

import (
	"github.com/china8036/gdht"
	"local/oauth2/lib"
)

const BLOCK = 16384

func main() {
	infoHash := "91deb3de3c09a300e2b987efc4fa8508bbd924dc"
	ih, _ := gdht.DecodeInfoHash(infoHash)
	address := "198.49.188.161:1498"
	gdht.GetMetaInfo(ih,lib.GenRandomString(20),address)
	}
