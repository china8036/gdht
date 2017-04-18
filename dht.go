package gdht

import (
	"net"
	"log"
	"sync"
	"math/rand"
	"time"
	"fmt"
)

const (
	KlStorePeriod = time.Minute //k桶定时保存时间
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
	Kl     *KbucketList
	wg     sync.WaitGroup
}

func New() (*DHT, error) {
	var nodeid string
	k, err := NewKrpc()
	if err != nil {
		return nil, err
	}
	var kl *KbucketList
	kl, err = ObtainStoreKbucketList(k.port)
	if err == nil && kl.Nodeid != "" {
		log.Println(kl.Nodeid)
		nodeid = kl.Nodeid
	} else {
		nodeid = string(RandNodeId())
		log.Println("obtain store kl err ", err)
		kl = NewKbucketList(nodeid)
	}
	k.NodeId = nodeid
	return &DHT{NodeId: nodeid, k: k, Kl: kl}, nil

}

//开启启动服务
func (d *DHT) Run() {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.loop()
	}()
	for _, saddr := range SupperNode {
		d.k.Ping(saddr, nil) //ping每个超级节点 他们回复时候就讲超级节点加入到自己的K桶
	}
	time.Sleep(time.Second * 10)
	d.FindNode(d.NodeId)

}

//主loop主要是消息处理
func (d *DHT) loop() {
	for {
		select {
		case p := <-d.k.packetChan:
			d.dealPacket(p)
			break
		case <-time.NewTimer(KlStorePeriod).C:
			log.Println("store Kl")
			go SaveKbucketList(d)
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
	case Error: //err回复处理
		log.Println("err ", p.r.E)
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
	d.Kl.UpdateOne(laddr, r.R.Id, d.k) //更新K桶 根据协议 其他节点的任何回应信息 都应该去检查更新K桶
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
		log.Println("receive not defined response ", r.R)
	}
}

//其他节点查询本节点请求
func (d *DHT) dealQuery(r responseType, laddr *net.UDPAddr) {
	if !IsNodeId(r.A.Id) {
		log.Println("ivalid query node id ", r.A.Id)
		return
	}
	//d.k.Ping("", laddr) //对方节点查询 主动ping下节点 通过后加入自己的K桶 网络上有很多坏节点 只查询其他节点 不对其他节点对其的查询做回应 这样的不加入K桶
	log.Println("some one query me", r.A.Id)
	switch r.Q {
	case Ping:
		d.k.ResponsePing(r, laddr)
		break
	case FindNode:                                        //查找节点的回复
		closetnodes := d.Kl.LookUpClosetNodes(r.A.Id) //查找最近的nodes信息返回
		d.k.ResponseFindNode(closetnodes, r, laddr)
		break
	case GetPeers: //查找peer的回复
		break
	case AnnouncePeer: //其他节点发布信息他有相应的文件下载信息 你可以存储
		log.Println("find info hash ")
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
		d.k.Ping("", enode.Addr) //对于可利用的node都要ping一下 尝试加入到自己的K桶中
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
	if minDistanceNode.Nodeid == reply_node.Nodeid { //最近的就是回复的节点 停止查询
		log.Println("find the closest node ", reply_node.Nodeid)
		log.Printf(" dis %x\n", NodeDistance(target, reply_node.Nodeid))
		return
	}
	closeNodes := FindMinDistanceNodes(FindNodeCloseNum, nodes, target) //选取最近的几个继续进行findnode操作
	//继续对每个返回的node执行find_node操作
	for _, cnode := range closeNodes {
		if cnode.Nodeid == d.NodeId { //过滤到自己
			continue
		}
		log.Println("continue find node ", target, " search in ", cnode.Nodeid)
		log.Printf(" dis %x\n", NodeDistance(target, cnode.Nodeid))
		d.k.FindNode(target, "", cnode.Addr) //继续对返回的节点进行find_node查询
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
		//if closet_node.Nodeid == reply_node.Nodeid { //最近的等于回复的节点 说明此节点如果没有相关peers信息再查就没必要了此路终止查询
		//	log.Println(reply_node.Nodeid, " closet node not found peers ", info_hash)
		//	log.Printf(" dis %x\n", NodeDistance(info_hash, reply_node.Nodeid))
		//	return
		//}
		for _, wait_get_peers_node := range nodes {
			if InfoFindPeers[info_hash][wait_get_peers_node.Nodeid] {
				log.Println(info_hash, " get peers skip ", wait_get_peers_node.Nodeid)
				continue
			}
			InfoFindPeers[info_hash][wait_get_peers_node.Nodeid] = true
			log.Printf(" continue search info_hash %s in %s node", info_hash, wait_get_peers_node.Nodeid)
			log.Printf(" dis %x\n", NodeDistance(info_hash, wait_get_peers_node.Nodeid))
			d.k.GetPeers(info_hash, "", wait_get_peers_node.Addr)
		}
	}
	if r.R.Values != nil { //发现peers 终止查询
		delete(InfoFindPeers, info_hash) //找到能链接的peer 删除记录
		addrs := ParseContactpeers(r.R.Values)
		if len(addrs) < 1 {
			log.Println("not found real peers addrs")
			return
		} //找到peers
		peer_id := string(RandNodeId())
		for _, addr := range addrs {
			log.Println(info_hash, " get_peers find Addr:", addr.IP.String(), addr.Port)
			go GetMetaInfo(info_hash,peer_id,fmt.Sprintf("%s:%d",addr.IP.String(),addr.Port))

		}

	}
	return

}

//dht查询节点 一般用于组建自己的K桶用
//从K桶中选8个最近的节点
func (d *DHT) FindNode(nodeid string) {
	nodes := d.Kl.LookUpClosetNodes(nodeid)
	if len(nodes) < 1 {
		log.Println("not found clost Nodes")
		return
	}
	for _, node := range nodes {
		d.k.FindNode(nodeid, "", node.Addr) //对最近的8个节点自行FindNode操作
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
		InfoFindPeers[info_hash] = make(map[string]bool) //暂时不做时间处理 所有时间对info_hash的搜索
		// 都算因为info_hash一样搜索的东西也一样
		// 前一个搜索出来 这一次也用同样的结果就可以了
	}
	nodes := d.Kl.LookUpClosetNodes(info_hash)
	if len(nodes) < 1 {
		log.Println("Getpeers not found clost Nodes")
		return
	}
	for _, node := range nodes {
		InfoFindPeers[info_hash][node.Nodeid] = true
		d.k.GetPeers(info_hash, "", node.Addr)
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
