package ecs

import (
	"errors"
	"sync"
	"time"
)

// ErrNoNodeAvailable is returned if all the nodes are blocked
var ErrNoNodeAvailable = errors.New("no ecs nodes available")

type node struct {
	host         string
	blockedUntil time.Time
}

// NewVdc ...
func NewVdc(id string, nodes []string) *Vdc {
	vdc := &Vdc{ID: id, Nodes: make([]node, len(nodes))}

	for i, n := range nodes {
		vdc.Nodes[i] = node{n, time.Time{}}
	}
	return vdc
}

// Vdc ...
type Vdc struct {
	sync.Mutex
	ID      string
	Nodes   []node
	current int
}

// NextAvailableNode ...
func (v *Vdc) NextAvailableNode() (string, error) {
	v.Lock()
	defer v.Unlock()
	now := time.Now()
	for i := 0; i < len(v.Nodes); i++ {

		v.current = (v.current + 1) % len(v.Nodes)
		if now.After(v.Nodes[v.current].blockedUntil) {
			return v.Nodes[v.current].host, nil
		}
	}
	return "", ErrNoNodeAvailable
}

// BlockNode is to block node in prefined duration
func (v *Vdc) BlockNode(host string, dur time.Duration) {
	for _, n := range v.Nodes {
		if n.host == host {
			v.Lock()
			defer v.Unlock()
			n.blockedUntil = time.Now().Add(dur)
			return
		}
	}
}

// NewEcs ...
func NewEcs(vdcs map[string]*Vdc) *Ecs {
	ecs := &Ecs{vdcs}
	return ecs
}

// Ecs ...
type Ecs struct {
	Vdcs map[string]*Vdc
}

// NextAvailableNode ...
func (e *Ecs) NextAvailableNode(vdcid string) (string, error) {
	if v, ok := e.Vdcs[vdcid]; ok {
		return v.NextAvailableNode()
	} else {
		for _, vdc := range e.Vdcs {
			return vdc.NextAvailableNode()
		}
	}
	return "", ErrNoNodeAvailable
}

// BlockNode is to block node in prefined duration
func (e *Ecs) BlockNode(host string, dur time.Duration) {
	for _, v := range e.Vdcs {
		v.BlockNode(host, dur)
	}
}
