package gdht

import (
	"encoding/json"
	"io/ioutil"
	"fmt"
	"os"
	"path"
	"log"
	"qiniupkg.com/x/errors.v7"
)

const (
	StoreFilePre       = "cache/%d.node"
	Max_log_size int64 = 2 * 1024 * 1024 //2M
)

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
	err = Save(d.k.port, bytes)
	if err != nil {
		return err
	}
	return nil
}

//解析kl
func ObtainStoreKbucketList(port int) (*KbucketList, error) {
	file := GetFile(port)
	bytes, err := ioutil.ReadFile(file)
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

func Save(port int, b []byte) error {
	file := GetFile(port)
	err := os.MkdirAll(path.Dir(file), os.ModePerm)
	if err != nil {
		return err
	}
	logfile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	n, err := logfile.Write(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return errors.New("not write all bytes")
	}
	return err
}

func GetFile(port int) string {
	return fmt.Sprintf("cache/%d.node", port)
}
