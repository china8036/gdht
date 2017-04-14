package gdht

import (
	"net"
	"log"
	"sync"
	"errors"
	"fmt"
)

var SupperNode = []string{
	"router.bittorrent.com:6881",
	"router.utorrent.com:6881",
	"router.magnets.im:6881",
	"dht.transmissionbt.com:6881"}

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
	for _, sn := range SupperNode {
		d.k.Ping(sn, nil)               //ping每个超级节点 他们回复时候就讲超级节点加入到自己的K桶
		d.k.FindNode(d.NodeId, sn, nil) //建立自己的K桶时候 不知道自己的K桶是否是空的 所以不能用d.FindNode() 这是改为用超级节点查询自己
	}

}

//主loop主要是消息处理
func (d *DHT) loop() {
	for {
		d.dealPacket(<-d.k.packetChan)
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
	if r.R.Id == d.NodeId { //对于自己的回应不处理
		log.Println("self response ", r)
		return
	}
	switch ParseResponseType(r.T) {
	case ResponsePing:
		d.kl.UpdateOne(laddr, r.R.Id, d.k) //更新K桶
		break
	case ResponseFindNode: //查找节点的回复
		d.dealFindNodeResponse(r, laddr)
		break
	case ResponseGetPeers: //查找peer的回复
		d.dealGetPeersResponse(r, laddr)
		break
	case ResponseAnnouncePeer: //通知对方这边有文件信息时候 对方的回应信息
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
	switch r.Q {
	case Ping:
		d.k.ResponsePing(r, laddr)
		break
	case FindNode: //查找节点的回复
		break
	case GetPeers: //查找peer的回复
		break
	case AnnouncePeer: //其他节点发布信息他有相应的文件下载信息 你可以存储
		break
	}
}

//其他节点回复的节点查询信息 比较返回的所有节点距目标节点的距离e1-n 并和回复的节点与目标节点的距离x比较 如果x最小测停止继续查找
func (d *DHT) dealFindNodeResponse(r responseType, laddr *net.UDPAddr) {
	target := GetFndNodeTarget(r.T)
	if !IsNodeId(target) {
		log.Println(target, "response target id ivalid", r.R.Id)
		return
	}
	nodes := d.k.ParseContactInformation(r.R.Nodes)
	if len(nodes) == 0 {
		return
	}
	for _, enode := range nodes {
		d.k.Ping("", enode.addr) //对于可利用的node都要ping一下 尝试加入到自己的K桶中
	}
	reply_node, err := NewNode(laddr, r.R.Id)
	if err != nil {
		log.Println(err)
		return
	}
	tem_nodes := append(nodes, *reply_node)
	minDistanceNode, _ := FindMinDistanceNode(tem_nodes, target)
	if minDistanceNode == nil {
		return
	}
	if minDistanceNode.nodeid == reply_node.nodeid { //最近的就是回复的节点 停止查询
		log.Println("find the closest node ", reply_node.nodeid)
		fmt.Printf(" dis %x\n", NodeDistance(target, reply_node.nodeid))
		return
	}
	closeNodes := FindMinDistanceNodes(FindNodeCloseNum, nodes, target) //选取最近的几个继续进行findnode操作
	//继续对每个返回的node执行find_node操作
	for _, cnode := range closeNodes {
		if cnode.nodeid == d.NodeId { //过滤到自己
			continue
		}
		log.Println("continue find node ", target, " search in ", cnode.nodeid)
		fmt.Printf(" dis %x\n", NodeDistance(target, cnode.nodeid))
		d.k.FindNode(target, "", cnode.addr) //继续对返回的节点进行find_node查询
	}

}

//处理GetPeer回应信息
func (d *DHT) dealGetPeersResponse(r responseType, laddr *net.UDPAddr) {
	//info_hash := GetGetPeesInfoHash(r.T)
	if r.R.Nodes != "" {
		nodes := d.k.ParseContactInformation(r.R.Nodes)
		log.Println("rec peers closest nodes", nodes)
	}
	if r.R.Values != nil {
		log.Println("find Values", r.R.Values)
	}
	return

}

//dht查询节点 一般用于组建自己的K桶用
//从K桶中选8个最近的节点
func (d *DHT) FindNode(nodeid string) {
	nodes := d.kl.LookUpClosetNodes(nodeid)
	if len(nodes) < 1 {
		log.Println("not found clost nodes")
		return
	}
	for _, node := range nodes {
		d.k.FindNode(nodeid, "", node.addr) //对最近的8个节点自行FindNode操作
	}

}

//等待
func (d *DHT) Wait() {
	d.wg.Wait()
}
