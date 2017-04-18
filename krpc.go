package gdht

import (
	"bytes"
	bencode "github.com/jackpal/bencode-go"
	"net"
	"log"
	"fmt"
	"sync"
	"encoding/binary"
	"encoding/hex"
	"time"
	"strings"
	"strconv"
)

const (
	P                    = "udp"
	BencodeFchr          = 'd'
	MaxAcceptLen         = 4096
	MaxPacket            = 100 //接受的包 chan长
	MaxQueryList         = 100 //发送的包 chan长
	BeginPort            = 6881
	EndPort              = 6891
	EcontactInfoLen      = 26 //紧凑型node返回信息 26个字节 20个字节nodeid 4个字节ip 2个字节端口
	EcontactPeerLen      = 6  //紧凑型peer信息 6分直接 ip+端口
	FindNodeCloseNum     = 8  //选最近的8个
	UdpWriteDeadline     = 1
	Query                = "q"
	Response             = "r"
	Error                = "e"
	Ping                 = "ping"
	FindNode             = "find_node"
	GetPeers             = "get_peers"
	AnnouncePeer         = "announce_peer"
	ResponsePing         = "ping"
	ResponseFindNode     = "find"
	ResponseGetPeers     = "peer"
	ResponseAnnouncePeer = "ance"
)

type getPeersResponse struct {
	// TODO: argh, values can be a string depending on the client (e.g: original bittorrent).
	Values []string "values"
	Id     string   "id"
	Nodes  string   "Nodes"
	Nodes6 string   "nodes6"
	Token  string   "token"
}

type answerType struct {
	Id       string   "id"
	Target   string   "target"
	InfoHash string  "info_hash" // should probably be a string.
	Port     int      "port"
	Token    string   "token"
}

// Generic stuff we read from the wire, not knowing what it is. This is as generic as can be.
type responseType struct {
	T string           "t"
	Y string           "y"
	Q string           "q"
	R getPeersResponse "r"
	E []string         "e"
	A answerType       "a"
	// Unsupported mainline extension for client identification.
	// V string(?)	"v"
}

// Message to be sent out in the wire. Must not have any extra fields.
type QueryMessage struct {
	T string                 "t"
	Y string                 "y"
	Q string                 "q"
	A map[string]interface{} "a"
}

type replyMessage struct {
	T string                 "t"
	Y string                 "y"
	R map[string]interface{} "r"
}

//error消息
type ErrorMessage struct {
	T string                 "t"
	Y string                 "y"
	E []interface{} "e"
}

type packetType struct {
	r     responseType
	raddr *net.UDPAddr
}

type queryPacket struct {
	tranId   int
	data     []byte
	raddr    *net.UDPAddr
	sendTime time.Time
}

type Krpc struct {
	conn        *net.UDPConn
	port        int
	NodeId      string
	packetChan  chan packetType
	wg          sync.WaitGroup
	queryList   chan queryPacket //等待发送的链接
	pendingList []*queryPacket   //已发送在等待回应的列表
	lastTranId  int
}

//监听udp如果失败则加1继续尝试
func NewKrpc() (*Krpc, error) {
	var lister net.PacketConn
	var err error

	i := BeginPort
	for i = BeginPort; i <= EndPort; i++ {
		lister, err = net.ListenPacket("udp", fmt.Sprintf(":%d", i))
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	fmt.Println(i)
	conn := lister.(*net.UDPConn)
	//conn.SetWriteDeadline(time.Now().Add(time.Second * UdpWriteDeadline)) //设置udp写入时间限制
	k := &Krpc{conn:     conn,
		port:        i,
		packetChan:  make(chan packetType, MaxPacket),
		queryList:   make(chan queryPacket, MaxQueryList),
		pendingList: make([]*queryPacket, MaxQueryList)}
	k.wg.Add(1)
	go func() {
		defer k.wg.Done()
		k.BeginAcceptMsg()
	}()
	k.wg.Add(1)
	go func() {
		defer k.wg.Done()
		k.BeginSendMsg()
	}()
	return k, nil
}

//ping一个地址
func (k *Krpc) Ping(addr string, laddr *net.UDPAddr) {
	laddr = GetLaddr(addr, laddr)
	if laddr == nil {
		return
	}
	query := QueryMessage{T: ResponsePing, Y: Query, Q: Ping, A: map[string]interface{}{"id": k.NodeId}}
	k.SendMsg(laddr, query)
}

//查询nodes
func (k *Krpc) FindNode(nodeid, addr string, laddr *net.UDPAddr) {
	laddr = GetLaddr(addr, laddr)
	if laddr == nil {
		return
	}
	query := QueryMessage{T: GenFindNodeT(nodeid), Y: Query, Q: FindNode, A: map[string]interface{}{"id": k.NodeId, "target": nodeid}}
	k.SendMsg(laddr, query)
}

//查询Peers信息
func (k *Krpc) GetPeers(ih, addr string, laddr *net.UDPAddr) {
	if len(ih) != NodeIdLen { //info_hash的长度 和 nodeid是一样的
		log.Println("info hash long err ", ih)
		return
	}
	laddr = GetLaddr(addr, laddr)
	if laddr == nil {
		return
	}
	query := QueryMessage{T: GenGetPeersT(ih), Y: Query, Q: GetPeers, A: map[string]interface{}{"id": k.NodeId, "info_hash": ih }}
	k.SendMsg(laddr, query)

}

//announce peer
func (k *Krpc) AnnouncePeer(ih, token string, port int, addr string, laddr *net.UDPAddr) {
	laddr = GetLaddr(addr, laddr)
	if laddr == nil {
		return
	}
	query := QueryMessage{
		T: ResponseAnnouncePeer,
		Y: "q",
		Q: AnnouncePeer, //"implied_port": 0,
		A: map[string]interface{}{"id": k.NodeId, "info_hash": ih, "port": port, "token": token}}
	k.SendMsg(laddr, query)

}

//回复ping
func (k *Krpc) ResponsePing(r responseType, laddr *net.UDPAddr) {
	log.Println("some one ping me", r.A.Id)
	reply := replyMessage{
		T: r.T,
		Y: Response,
		R: map[string]interface{}{"id": k.NodeId},
	}
	k.SendMsg(laddr, reply)
}

//回应node查找信息直接返回最近的8个节点即可
func (k *Krpc) ResponseFindNode(nodes []node, response responseType, laddr *net.UDPAddr) {
	var r []byte
	for _, en := range nodes {
		ips := en.Addr.IP.String()
		ipslice := strings.Split(ips, ".")
		var ipbyte [4]byte
		for i, es := range ipslice {
			esi, _ := strconv.Atoi(es)
			ipbyte[i] = byte(esi)
		}

		tmp := int16(en.Addr.Port)
		bytesBuffer := bytes.NewBuffer([]byte{})
		binary.Write(bytesBuffer, binary.BigEndian, tmp)
		all := append([]byte(en.Nodeid), ipbyte[0:]...)
		all = append(all, bytesBuffer.Bytes()...)
		r = append(r, all...)
	}
	log.Println("response find node len ", len(r))
	reply := replyMessage{
		T: response.T,
		Y: Response,
		R: map[string]interface{}{"id": k.NodeId, "token": "aoeusnth", "Nodes": string(r)},
	}
	k.SendMsg(laddr, reply)

}

//回应无效的请求
func (k *Krpc) ResponseInvalid(r responseType, laddr *net.UDPAddr) {
	error := ErrorMessage{
		T: r.T,
		Y: "e",
		E: []interface{}{201, "this type not defind, Protocol Error, such as a malformed packet, invalid arguments, or bad token"},
	}
	k.SendMsg(laddr, error)

}

// sendMsg bencodes the data in 'query' and sends it to the remote node.
func (k *Krpc) SendMsg(raddr *net.UDPAddr, query interface{}) {
	var b bytes.Buffer
	if err := bencode.Marshal(&b, query); err != nil {
		return
	}
	k.queryList <- queryPacket{data: b.Bytes(), raddr: raddr, tranId: k.lastTranId} //写入处理队列 来均衡每一处理的时间
}

//开始发送信息
func (k *Krpc) BeginSendMsg() {
	for {
		query := <-k.queryList
		if n, err := k.conn.WriteToUDP(query.data, query.raddr); err != nil {
			log.Println(err, GetDefaultTrace())
		} else {
			log.Println("write to ", query.raddr, string(query.data), n)
		}
		query.sendTime = time.Now()
		k.pendingList[query.tranId] = &query
	}

}

//回应消息
func (k *Krpc) BeginAcceptMsg() {
	for {
		var accept_data = make([]byte, MaxAcceptLen)
		n, addr, err := k.conn.ReadFromUDP(accept_data)
		if err != nil {
			log.Println("udp read error", err)
			return
		}
		if n == MaxAcceptLen {
			log.Println("accept msg too long drop some", accept_data)
		}
		accept_data = accept_data[0:n]
		if accept_data[0] != BencodeFchr {
			log.Println("receive data is not bencoded", accept_data)
			continue
		}
		var response responseType
		err1 := DecondeAcceptMsg(accept_data, &response)
		if err1 != nil {
			continue
		}
		k.packetChan <- packetType{response, addr}
	}

}

//解析紧凑型信息为node切边
func (k *Krpc) ParseContactNodes(contactInfo string) []node {
	var nodes []node
	cl := len(contactInfo)
	if cl < EcontactInfoLen || cl%EcontactInfoLen != 0 { //整除
		return nodes
	}
	max := int(cl / EcontactInfoLen)
	binfo := []byte(contactInfo)
	for i := 0; i < max; i++ {
		b := EcontactInfoLen * i
		hash := binfo[b:b+20]
		ip := binfo[b+20:b+24]
		bytesBuffer := bytes.NewBuffer(binfo[(b + 24):b+26])
		var port uint16
		binary.Read(bytesBuffer, binary.BigEndian, &port)
		newnode := node{Nodeid: string(hash), Addr: &net.UDPAddr{IP: ip, Port: int(port)}}
		nodes = append(nodes, newnode)
	}
	return nodes

}

//解析peers的紧凑信息 peer返回的ip port为 tcp 协议
func ParseContactpeers(contactPeers []string) []net.TCPAddr {
	var addrs []net.TCPAddr
	for _, cp := range contactPeers {
		info := []byte(cp)
		ip := net.IPv4(info[0], info[1], info[2], info[3])
		port := int((uint16(info[4]) << 8) | uint16(info[5]))
		addrs = append(addrs, net.TCPAddr{IP: ip, Port: int(port)})
	}
	return addrs
}

func (k *Krpc) Wait() {
	k.wg.Wait()
}

//解析接受的记录
func DecondeAcceptMsg(accept_data []byte, response *responseType) error {
	defer func() {
		if x := recover(); x != nil {
			log.Println(x)
		}
	}()
	if e2 := bencode.Unmarshal(bytes.NewBuffer(accept_data), &response); e2 != nil {
		return e2
	}
	return nil
}

//尝试使用多种地址 优先使用laddr
func GetLaddr(addr string, laddr *net.UDPAddr) *net.UDPAddr {
	if laddr == nil {
		raddr, err := net.ResolveUDPAddr(P, addr)
		if err != nil {
			log.Println(err)
			return nil
		}
		laddr = raddr
	}
	return laddr
}

//解析返回信息
func ParseResponseType(t string) string {
	if len(t) < 4 {
		return t
	}
	bt := []byte(t)
	ty := string(bt[0:4])
	return ty
}

//自定义findnode的t值 便于信息回复时候得到target
func GenFindNodeT(target string) string {
	return fmt.Sprintf("%s%s", ResponseFindNode, target)
}

//处理findnode的t值得到target
func GetFndNodeTarget(t string) string {
	l := len(ResponseFindNode)
	if len(t) < l {
		return t
	}
	return t[l:]
}

//处理getpeers的头部使其记录infohash值
func GenGetPeersT(info_hash string) string {
	return fmt.Sprintf("%s%s", ResponseGetPeers, info_hash)
}

//处理getpeers的t值得到infohash
func GetGetPeesInfoHash(t string) string {
	l := len(ResponseGetPeers)
	if len(t) < l {
		return t
	}
	return t[l:]
}

// DecodeInfoHash transforms a hex-encoded 20-characters string to a binary
// infohash.
func DecodeInfoHash(x string) (b string, err error) { //20位hash有时候是不可见字符
	var h []byte
	h, err = hex.DecodeString(x)
	return string(h), err
}

// DecodeInfoHash transforms a hex-encoded 20-characters string to a binary
// infohash.
func EncodeInfoHash(x string) string { //20位hash有时候是不可见字符 encode后转换为可见字符
	var h []byte
	hex.Encode(h, []byte(x))
	return string(h)
}
