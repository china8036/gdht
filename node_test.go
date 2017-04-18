package gdht

import (
	"testing"
	"net"
)

func TestFunc(t *testing.T) {
	raddr, err := net.ResolveUDPAddr(P, "127.0.0.1:6881")
	if err != nil {
		t.Error(err)
	}
	node1, err1 := NewNode(raddr, "aaaaaaaaaaaaaaaaaaaa")
	if err1 != nil {
		t.Error(err1)
	}
	node2, err2 := NewNode(raddr, "bbbbbbbbbbbbbbbbbbbb")
	if err2 != nil {
		t.Error(err2)
	}
	nodes := make([]node, 2)
	nodes[0] = *node1
	nodes[1] = *node2
	target := "cccccccccccccccccccc"
	node, index := FindMinDistanceNode(nodes, target)
	if index != 1 {
		t.Fail()
	}
	if node.Nodeid != node2.Nodeid {
		t.Fail()
	}
	minNodes := FindMinDistanceNodes(1, nodes, target)
	if len(minNodes) != 1 {
		t.Fail()
	}
	minNodes = FindMinDistanceNodes(3, nodes, target)
	if len(minNodes) != 2 {
		t.Fail()
	}

}
