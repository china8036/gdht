package main

import (
	"github.com/china8036/gdht"
	"local/oauth2/lib"
)

const BLOCK = 16384

func main() {
	infoHash := "2864cc1fb7ac6137ff6bfe4a091625fd607cdfac"
	ih, _ := gdht.DecodeInfoHash(infoHash)
	address := "94.156.123.245:1033"
	gdht.GetMetaInfo(ih,lib.GenRandomString(20),address)
	}
