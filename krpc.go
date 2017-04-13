package gdht

import (
	"bytes"
	bencode "github.com/jackpal/bencode-go"
	"net"
	"log"
	"fmt"
	"sync"
)

const (
	P            = "udp"
	BencodeFchr  = 'd'
	MaxAcceptLen = 4096
	MaxPacket    = 5
	BeginPort    = 6881
	EndPort      = 6891
	Query        = "q"
	Response     = "r"
	Ping         = "ping"
	FindNode     = "find_node"
	GetPeers     = "get_peers"
	AnnouncePeer = "announce_peer"
)

type getPeersResponse struct {
	// TODO: argh, values can be a string depending on the client (e.g: original bittorrent).
	Values []string "values"
	Id     string   "id"
	Nodes  string   "nodes"
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

type packetType struct {
	r     responseType
	raddr *net.UDPAddr
}

type Krpc struct {
	conn       *net.UDPConn
	NodeId     string
	packetChan chan packetType
	wg         sync.WaitGroup
}

//监听udp如果失败则加1继续尝试
func NewKrpc(nodeid string) (*Krpc, error) {
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
	k := &Krpc{conn: conn, NodeId: nodeid, packetChan: make(chan packetType, MaxPacket)}
	k.wg.Add(1)
	go func() {
		defer k.wg.Done()
		k.BeginAcceptMsg()
	}()
	return k, nil
}

//ping一个地址
func (k *Krpc) Ping(addr string) {
	raddr, err := net.ResolveUDPAddr(P, addr)
	if err != nil {
		log.Println(err)
		return
	}
	query := QueryMessage{T: Ping, Y: Query, Q: Ping, A: map[string]interface{}{"id": k.NodeId}}
	k.SendMsg(raddr, query)
}

func (k *Krpc) FindNode(addr, nodeid string) {
	raddr, err := net.ResolveUDPAddr(P, addr)
	if err != nil {
		log.Println(err)
		return
	}
	query := QueryMessage{T: FindNode, Y: Query, Q: FindNode, A: map[string]interface{}{"id": k.NodeId, "target": nodeid}}
	k.SendMsg(raddr, query)
}

func (k *Krpc) ResponsePing(r responseType, laddr *net.UDPAddr) {
	log.Println("some one ping me", r.A.Id)
	reply := replyMessage{
		T: r.T,
		Y: Response,
		R: map[string]interface{}{"id": k.NodeId},
	}
	k.SendMsg(laddr, reply)
}

// sendMsg bencodes the data in 'query' and sends it to the remote node.
func (k *Krpc) SendMsg(raddr *net.UDPAddr, query interface{}) {
	var b bytes.Buffer
	if err := bencode.Marshal(&b, query); err != nil {
		return
	}
	if n, err := k.conn.WriteToUDP(b.Bytes(), raddr); err != nil {
		log.Println(err)
	} else {
		log.Println("write to ", raddr, string(b.Bytes()), n)
	}
	return
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
		if e2 := bencode.Unmarshal(bytes.NewBuffer(accept_data), &response); e2 != nil {
			log.Println(e2)
			continue
		}
		k.packetChan <- packetType{response, addr}
	}

}

func (k *Krpc) Wait() {
	k.wg.Wait()
}
