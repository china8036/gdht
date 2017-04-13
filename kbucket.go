package gdht

import (
	"math"
	"net"
	"sync"
	"strings"
	"log"
	"time"
)

const (
	EachKbMaxLen = 8
	MaxKbNum     = 161
	CheckPingSecond  = 5
)

type KbucketList struct {
	nodeid   string
	kbuckets []kbucket
}

//新k桶列表
func NewKbucketList(nodeid string) *KbucketList {
	var kbuckets [MaxKbNum]kbucket
	return &KbucketList{nodeid, kbuckets[0:]}
}

//一个K桶
type kbucket struct {
	lock sync.Mutex
	nodes []node
	first_locked bool//第一个节点是否锁定
	fight_node node//争夺K桶的节点 如果一定时间第一个节点未返回ping信息 第一个将被抛掉 此节点加入K桶尾部
}

//增加一个node此方法只能在ping此节点被回应后调用
func (kl *KbucketList) UpdateOne(addr *net.UDPAddr, nodeid string,k *Krpc) {
	if kl.kbuckets == nil {
		return
	}
	log.Println(kl.kbuckets)
	newNode := node{addr: addr, nodeid: nodeid}
	dis := NodeDistance(kl.nodeid, nodeid)
	index := FindDistanceIndex(dis)
	if kl.kbuckets[index].nodes == nil{//空K桶
		kl.kbuckets[index].nodes = make([]node,0)
		kl.kbuckets[index].nodes = append(kl.kbuckets[index].nodes,newNode)
		log.Println(index," init and add first node ")
		return
	}
	for i, v := range kl.kbuckets[index].nodes { //查询是否已经存在
		if strings.EqualFold(v.addr.IP.String(),newNode.addr.IP.String()){ //对应的Ip存在K桶中 则更新信息并移到尾部(表示持续在线 在线的可能性更大 提升权重)
			if index==0 && kl.kbuckets[index].first_locked{//此信息为检查第一个节点是否正常的信息 不正常就抛掉 走到这步说明此节点正常
				kl.kbuckets[index].first_locked = false//接触锁定
			}
			kl.kbuckets[index].lock.Lock()//锁定
			kl.kbuckets[index].nodes = append(kl.kbuckets[index].nodes[:i], kl.kbuckets[index].nodes[(i + 1):]...)
			kl.kbuckets[index].nodes = append(kl.kbuckets[index].nodes, newNode) //这两步是把此节点移到尾部
			kl.kbuckets[index].lock.Unlock()//解锁
			log.Println(index," update ",i," to bottom")
			return
		}
	}
	//此为不存在的情况
	if len(kl.kbuckets[index].nodes) <= EachKbMaxLen {//数量不足时候 直接插入到尾部
		kl.kbuckets[index].lock.Lock()//锁定
		kl.kbuckets[index].nodes = append(kl.kbuckets[index].nodes, newNode)
		kl.kbuckets[index].lock.Unlock()//解锁
		log.Println(index," add new node to bottom")
		return
	}
	//如果K桶数量充足 则ping第一个如果有回应 则抛弃新的同时把第一个放到尾部否则替换第一个
	if kl.kbuckets[index].first_locked{//如果有节点正在和第一个节点争夺 其他节点进来是直接抛弃掉
		return
	}
	kl.kbuckets[index].first_locked = true
        k.NodeCheckPing(kl.kbuckets[index].nodes[0].addr.String())
	go func(){
		<-time.NewTimer(time.Second * CheckPingSecond).C//定时5秒 如果还在锁定状态则抛掉第一个节点加入此节点到尾部
		if !kl.kbuckets[index].first_locked{//已经被解除 说明第一个节点是好的  则直接抛掉此节点
			return
		}
		kl.kbuckets[index].lock.Lock()//锁定
		kl.kbuckets[index].nodes = kl.kbuckets[index].nodes[1:]//抛掉第一个节点
		kl.kbuckets[index].nodes = append(kl.kbuckets[index].nodes,newNode)
		kl.kbuckets[index].lock.Unlock()//锁定
	}()



}

//找出这个距离在K桶中的索引值
func FindDistanceIndex(d1 string) int {
	var b [20]byte
	for i, _ := range b {
		b[i] = 0x00
	}
	for m := 0; m < MaxKbNum-1; m++ {
		if d1 >= FindIndexDistanceLimit(m) && d1 <= FindIndexDistanceLimit(m+1) {
			return m
		}
	}
	return MaxKbNum-1

}

//查出第i个k桶的起始距离
func FindIndexDistanceLimit(i int) (s string) {
	i = i % 160 //确认不出界
	var b [20]byte
	for index, _ := range b {
		b[index] = 0x00
	}
	mi := int(i % 8)
	index := 19 - int(i/8) //
	b[index] = byte(int(math.Pow(2, float64(mi))))
	return string(b[0:])
}
