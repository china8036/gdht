package gdht

import (
	"bytes"
	bencode "github.com/jackpal/bencode-go"
	"net"
	"qiniupkg.com/x/log.v7"
	"fmt"
	"sync"
)

const (
	MaxAcceptLen = 4096
	MaxPacket    = 5
	BeginPort    = 6881
	EndPort      = 6891
)

type packetType struct {
	b     []byte
	raddr *net.UDPAddr
}

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

type Krpc struct {
	conn       *net.UDPConn
	NodeId     string
	packetChan chan packetType
	wg         sync.WaitGroup
}

//监听udp如果失败则加1继续尝试
func New(nodeid string) (*Krpc, error) {
	var lister net.PacketConn
	var err error
	for i := BeginPort; i <= EndPort; i++ {
		lister, err = net.ListenPacket("udp", fmt.Sprintf(":%d", i))
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	conn := lister.(*net.UDPConn)
	k := &Krpc{conn: conn, NodeId: nodeid, packetChan: make(chan packetType, MaxPacket)}
	k.wg.Add(1)
	go func() {
		defer  k.wg.Done()
		k.DealReceivePacket()
	}()
        k.wg.Add(1)
	go func() {
		defer  k.wg.Done()
		k.ResponseMsg()
	}()
	return k, nil
}

// sendMsg bencodes the data in 'query' and sends it to the remote node.
func (k *Krpc) SendMsg(raddr net.UDPAddr, query interface{}) {
	var b bytes.Buffer
	if err := bencode.Marshal(&b, query); err != nil {
		return
	}
	if n, err := k.conn.WriteToUDP(b.Bytes(), &raddr); err != nil {
		log.Error(err)
	} else {
		log.Infof("write to %v:%v[%d]", raddr, string(b.Bytes()), n)
	}
	return
}

//回应消息
func (k *Krpc) ResponseMsg() {
	for {
		var accept_data = make([]byte, MaxAcceptLen)
		n, addr, err := k.conn.ReadFromUDP(accept_data)
		if err != nil {
			log.Info("udp read error %v", err)
			return
		}
		if n == MaxAcceptLen {
			log.Infof("accept msg too long drop some %v", accept_data)
		}
		accept_data = accept_data[0:n]
		k.packetChan <- packetType{accept_data, addr}
	}

}

//处理接受的请求包
func (k *Krpc) DealReceivePacket() {
	for {
		p := <-k.packetChan
		if p.b[0] != 'd' {
			// Malformed DHT packet. There are protocol extensions out
			// there that we don't support or understand.
			log.Infof("receive ot bencode data %v", p)
			return
		}
		fmt.Println(string(p.b))
		var response responseType
		// The calls to bencode.Unmarshal() can be fragile.
		defer func() {
			if x := recover(); x != nil {
				log.Error(x)
			}
		}()
		if e2 := bencode.Unmarshal(bytes.NewBuffer(p.b), &response); e2 != nil {
			log.Error(e2)
			return
		}
	}
}


func (k *Krpc) Wait(){
	k.wg.Wait()
}