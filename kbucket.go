package gdht

import (
	"math"
	"net"
	"sync"
	"strings"
	"fmt"
	"log"
)

const (
	EachKbMaxLen = 8
	MaxKbNum     = 161
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
		return
	}
	for i, v := range kl.kbuckets[index].nodes { //查询是否已经存在
		fmt.Println(v.addr.IP.String())
		if strings.EqualFold(v.addr.IP.String(),newNode.addr.IP.String()){ //对应的Ip存在K桶中 则更新信息并移到尾部(表示持续在线 在线的可能性更大 提升权重)
			kl.kbuckets[index].lock.Lock()//锁定
			kl.kbuckets[index].nodes = append(kl.kbuckets[index].nodes[:i], kl.kbuckets[index].nodes[(i + 1):]...)
			kl.kbuckets[index].nodes = append(kl.kbuckets[index].nodes, newNode) //这两步是把此节点移到尾部
			kl.kbuckets[index].lock.Unlock()//解锁
			return
		}
	}
	//此为不存在的情况
	if len(kl.kbuckets[index].nodes) <= EachKbMaxLen {//数量不足时候 直接插入到尾部
		kl.kbuckets[index].lock.Lock()//锁定
		kl.kbuckets[index].nodes = append(kl.kbuckets[index].nodes, newNode)
		kl.kbuckets[index].lock.Unlock()//解锁
		return
	}
	//如果K桶数量充足 则ping第一个如果有回应 则抛弃新的同时把第一个放到尾部否则替换第一个
        k.Ping(kl.kbuckets[index].nodes[0].addr.String())

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
