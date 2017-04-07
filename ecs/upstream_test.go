package ecs

import (
	"sort"
	"testing"
	"time"
)

// TestVdc ...
func TestVdc(t *testing.T) {
	IPs := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}
	id := "vdc1"
	// New
	vdc := NewVdc(id, IPs)
	AssertNotEqualFatal(t, nil, vdc, "")
	AssertEqual(t, []node{
		node{"1.1.1.1", time.Time{}},
		node{"2.2.2.2", time.Time{}},
		node{"3.3.3.3", time.Time{}},
		node{"4.4.4.4", time.Time{}},
	}, vdc.Nodes, "")
	// NextAvailableNode
	for i := 0; i < 10; i++ {
		s, err := vdc.NextAvailableNode()
		AssertEqual(t, nil, err, "")
		idx := sort.SearchStrings(IPs, s)
		AssertEqual(t, true, idx < len(IPs) && IPs[idx] == s, "")
	}
	// BlockNode
	vdc.BlockNode("2.2.2.2", time.Second)
	for i := 0; i < 100; i++ {
		s, _ := vdc.NextAvailableNode()
		AssertNotEqual(t, "2.2.2.2", s, "")
	}
}

// TestEcs ...
func TestEcs(t *testing.T) {
	IPs1 := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}
	id1 := "vdc1"
	IPs2 := []string{"5.5.5.5", "6.6.6.6", "7.7.7.7", "8.8.8.8"}
	id2 := "vdc2"
	ecs := NewEcs(map[string]*Vdc{
		id1: NewVdc(id1, IPs1),
		id2: NewVdc(id2, IPs2),
	})
	AssertNotEqualFatal(t, nil, ecs, "")
	// NextAvailableNode
	for i := 0; i < 10; i++ {
		s, err := ecs.NextAvailableNode(id2)
		AssertEqual(t, nil, err, "")
		idx := sort.SearchStrings(IPs2, s)
		AssertEqual(t, true, idx < len(IPs2) && IPs2[idx] == s, "")
	}
	// NextAvailableNode
	for i := 0; i < 10; i++ {
		s, err := ecs.NextAvailableNode("")
		AssertEqual(t, nil, err, "")
		idx1 := sort.SearchStrings(IPs1, s)
		idx2 := sort.SearchStrings(IPs2, s)
		AssertEqual(t, true, idx1 < len(IPs1) && IPs1[idx1] == s || idx2 < len(IPs2) && IPs2[idx2] == s, "")
	}
	// BlockNode
	ecs.BlockNode("2.2.2.2", time.Second)
	for i := 0; i < 100; i++ {
		s, _ := ecs.NextAvailableNode("")
		AssertNotEqual(t, "2.2.2.2", s, "")
	}
}
