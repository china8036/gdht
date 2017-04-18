package gdht

import (
	"encoding/json"
	"io/ioutil"
	"fmt"
	"os"
	"path"
	"log"
)

const StoreFilePre = "cache/%d.node"

//保存kl
func SaveKbucketList(d *DHT) error {
	err := os.MkdirAll(path.Dir(StoreFilePre), os.ModePerm)
	if err != nil {
		return err
	}
	bytes, err := json.Marshal(*d.Kl)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fmt.Sprintf(StoreFilePre, d.k.port), bytes, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

//解析kl
func ObtainStoreKbucketList(port int, ) (*KbucketList, error) {
	bytes, err := ioutil.ReadFile(fmt.Sprintf(StoreFilePre, port))
	if err != nil {
		return nil, err
	}
	var kl KbucketList
	err = json.Unmarshal(bytes, &kl)
	if err != nil {
		log.Println("unm err ", err)
		return nil, err
	}
	return &kl, nil

}
