package gdht

import (
	"net"
	"log"
	"qiniupkg.com/x/errors.v7"
	"sync"
)

type DHT struct {
	NodeId string
	k      *Krpc
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
	return &DHT{NodeId: nodeid, k: k}, nil

}

//开启启动服务
func (d *DHT) Run() {
	d.wg.Add(1)
	go func(){
		defer  d.wg.Done()
		d.loop()
	}()
	d.k.Ping("router.bittorrent.com:6881")
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
	log.Println(r.T, " response", r.R)
	switch r.T {
	case Ping:
		dis := HashDistance(d.NodeId,r.R.Id)
		FindIndex(dis)
		break
	case FindNode:
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
	switch r.T {
	case Ping:
		d.k.ResponsePing(r, laddr)
		break
	}
}
