package gdht

import (
	"net"
	"log"
	"sync"
	"errors"
	"math/rand"
)

var SupperNode = []string{
	"router.bittorrent.com:6881",
	"router.utorrent.com:6881",
	"router.magnets.im:6881",
	"dht.transmissionbt.com:6881"}

var InfoFindPeers map[string]map[string]bool

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
	for _, saddr := range SupperNode {
		d.k.Ping(saddr, nil)               //ping每个超级节点 他们回复时候就讲超级节点加入到自己的K桶
		d.k.FindNode(d.NodeId, saddr, nil) //建立自己的K桶时候 不知道自己的K桶是否是空的 所以不能用d.FindNode() 这是改为用超级节点查询自己
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
	d.kl.UpdateOne(laddr, r.R.Id, d.k) //更新K桶 根据协议 其他节点的任何回应信息 都应该去检查更新K桶
	switch ParseResponseType(r.T) {
	case ResponsePing:
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
		log.Println("receive not defined response ", r.Q)
	}
}

//其他节点查询本节点请求
func (d *DHT) dealQuery(r responseType, laddr *net.UDPAddr) {
	if !IsNodeId(r.A.Id) {
		log.Println("ivalid query node id ", r.A.Id)
		return
	}
	d.k.Ping("", laddr) //对方节点查询 主动ping下节点 通过后加入自己的K桶 网络上有很多坏节点 只查询其他节点 不对其他节点对其的查询做回应 这样的不加入K桶
	log.Println("some one query me", r.A.Id)
	switch r.Q {
	case Ping:
		d.k.ResponsePing(r, laddr)
		break
	case FindNode:                                        //查找节点的回复
		closetnodes := d.kl.LookUpClosetNodes(r.A.Id) //查找最近的nodes信息返回
		d.k.ResponseFindNode(closetnodes, r, laddr)
		break
	case GetPeers: //查找peer的回复
		break
	case AnnouncePeer: //其他节点发布信息他有相应的文件下载信息 你可以存储
		log.Println("find new info hash ", r.A.InfoHash)
		break
	default:
		d.k.ResponseInvalid(r, laddr)
	}

}

//其他节点回复的节点查询信息 比较返回的所有节点距目标节点的距离e1-n 并和回复的节点与目标节点的距离x比较 如果x最小测停止继续查找
func (d *DHT) dealFindNodeResponse(r responseType, laddr *net.UDPAddr) {
	target := GetFndNodeTarget(r.T)
	if !IsNodeId(target) {
		log.Println(target, "response target id ivalid", r.R.Id)
		return
	}
	nodes := d.k.ParseContactNodes(r.R.Nodes)
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
		log.Printf(" dis %x\n", NodeDistance(target, reply_node.nodeid))
		return
	}
	closeNodes := FindMinDistanceNodes(FindNodeCloseNum, nodes, target) //选取最近的几个继续进行findnode操作
	//继续对每个返回的node执行find_node操作
	for _, cnode := range closeNodes {
		if cnode.nodeid == d.NodeId { //过滤到自己
			continue
		}
		log.Println("continue find node ", target, " search in ", cnode.nodeid)
		log.Printf(" dis %x\n", NodeDistance(target, cnode.nodeid))
		d.k.FindNode(target, "", cnode.addr) //继续对返回的节点进行find_node查询
	}

}

//处理GetPeer回应信息
func (d *DHT) dealGetPeersResponse(r responseType, laddr *net.UDPAddr) {
	info_hash := GetGetPeesInfoHash(r.T)
	if InfoFindPeers[info_hash] == nil { //已被销毁 或其他异常情况 已被销毁说明已经查询到
		return
	}
	reply_node, err := NewNode(laddr, r.R.Id)
	if err != nil {
		log.Println(err)
		return
	}
	if r.R.Nodes != "" { //没有返现时候返回的最近node信息
		nodes := d.k.ParseContactNodes(r.R.Nodes)
		if len(nodes) < 1 { //没有返回终止此路查询
			return
		}
		tmp_nodes := append(nodes, *reply_node)
		closet_node, _ := FindMinDistanceNode(tmp_nodes, info_hash) //找到最近的node
		if closet_node == nil {
			return
		}
		if closet_node.nodeid == reply_node.nodeid { //最近的等于回复的节点 说明此节点如果没有相关peers信息再查就没必要了此路终止查询
			log.Println(reply_node.nodeid, " closet node not found peers ", info_hash)
			log.Printf(" dis %x\n", NodeDistance(info_hash, reply_node.nodeid))
			return
		}
		for _, wait_get_peers_node := range nodes {
			if InfoFindPeers[info_hash][wait_get_peers_node.nodeid] {
				log.Println(info_hash, " get peers skip ", wait_get_peers_node.nodeid)
				continue
			}
			InfoFindPeers[info_hash][wait_get_peers_node.nodeid] = true
			log.Printf(" continue search info_hash %s in %s node", info_hash, wait_get_peers_node.nodeid)
			log.Printf(" dis %x\n", NodeDistance(info_hash, wait_get_peers_node.nodeid))
			d.k.GetPeers(info_hash, "", wait_get_peers_node.addr)
		}
	}
	if r.R.Values != nil { //发现peers 终止查询

		addrs := ParseContactpeers(r.R.Values)
		if len(addrs) < 1 {
			log.Println("not found real peers addrs")
			return
		}
		for _, addr := range addrs {
			log.Println(info_hash, " get_peers find addr:", addr.IP.String(),addr.Port)

		}
		delete(InfoFindPeers, info_hash) //找到peers 删除记录
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

//发起getpeers操作
func (d *DHT) GetPeers(encode_info_hash string) {
	info_hash, err := DecodeInfoHash(encode_info_hash)
	if err != nil {
		log.Println("info_hash decode err ", err)
		return
	}
	if InfoFindPeers == nil {
		InfoFindPeers = make(map[string]map[string]bool)
	}
	if InfoFindPeers[info_hash] == nil {
		InfoFindPeers[info_hash] = make(map[string]bool) //暂时不做时间处理 所有时间对info_hash的搜索 都算 先做调试用
	}
	nodes := d.kl.LookUpClosetNodes(info_hash)
	if len(nodes) < 1 {
		log.Println("Getpeers not found clost nodes")
		return
	}
	for _, node := range nodes {
		InfoFindPeers[info_hash][node.nodeid] = true
		d.k.GetPeers(info_hash, "", node.addr)
	}
}

//等待
func (d *DHT) Wait() {
	d.wg.Wait()
}

//随机生成nodeid
func RandNodeId() []byte {
	b := make([]byte, 20)
	rand.Read(b)
	return b
}
