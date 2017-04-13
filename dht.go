package gdht

import (
	"net"
	"log"
	"qiniupkg.com/x/errors.v7"
	"sync"
	"time"
	"fmt"
)

type DHT struct {
	NodeId string
	k      *Krpc
	kl     *KbucketList
	wg     sync.WaitGroup
}

func New(nodeid string) (*DHT, error) {
	if len(nodeid) != 20 {
		return nil, errors.New("ivalid node id")
	}
	k, err := NewKrpc(nodeid)
	if err != nil {
		return nil, err
	}
	kl := NewKbucketList(nodeid)
	return &DHT{NodeId: nodeid, k: k, kl: kl}, nil

}

//开启启动服务
func (d *DHT) Run() {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.loop()
	}()
	d.k.Ping("router.bittorrent.com:6881")
	time.Sleep(time.Second * 2)
	d.k.Ping("router.utorrent.com:6881")
	time.Sleep(time.Second * 2)
	d.k.Ping("dht.transmissionbt.com:6881")
	time.Sleep(time.Second * 2)
	d.k.Ping("router.bittorrent.com:6881")
	time.Sleep(time.Second * 2)
	d.k.FindNode("router.bittorrent.com:6881", d.NodeId)
	d.wg.Wait()

}

//主loop主要是消息处理
func (d *DHT) loop() {
	for {
		select {
		case p := <-d.k.packetChan:
			d.dealPacket(p)
			break
		}
	}
}

//处理接受的请求包
func (d *DHT) dealPacket(p packetType) {
	switch p.r.Y {
	case Response:
		d.dealResponse(p.r, p.raddr)
		break
	case Query:
		d.dealQuery(p.r, p.raddr)
		break
	default:
		log.Println("not defined", p)

	}
}

//查询的回应信息
func (d *DHT) dealResponse(r responseType, laddr *net.UDPAddr) {
	if !IsNodeId(r.R.Id) {
		log.Println("ivalid response node id ", r.R.Id)
		return
	}
	log.Println(r.T, " response", r.R.Id)
	switch r.T {
	case Ping:
		d.kl.UpdateOne(laddr, r.R.Id, d.k)
		break
	case FindNode:
		d.dealFindNodeResponse(r)
		break
	case GetPeers:
		break
	case AnnouncePeer:
		break
	default:
		log.Println("receive not defined response", r)
	}
}

//其他节点查询本节点请求
func (d *DHT) dealQuery(r responseType, laddr *net.UDPAddr) {
	if !IsNodeId(r.A.Id) {
		log.Println("ivalid query node id ", r.A.Id)
		return
	}
	log.Println(r.T, " response", r.A.Id)
	switch r.T {
	case Ping:
		d.k.ResponsePing(r, laddr)
		break
	}
}

//其他节点回复的节点查询信息
func (d *DHT) dealFindNodeResponse(r responseType) {
	nodes := d.k.ParseContactInformation(r.R.Nodes)
	if len(nodes) == 0 {
		return
	}
	fmt.Println(nodes)

}
