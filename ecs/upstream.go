package ecs

import (
	"errors"
	"math/rand"
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
	ID    string
	Nodes []node
}

// NextAvailableNode ...
func (v *Vdc) NextAvailableNode() (string, error) {
	v.Lock()
	now := time.Now()
	candidates := make([]node, 0)
	for _, n := range v.Nodes {
		if now.After(n.blockedUntil) {
			candidates = append(candidates, n)
		}
	}
	v.Unlock()
	if len(candidates) == 0 {
		return "", ErrNoNodeAvailable
	}
	return candidates[rand.Intn(len(candidates))].host, nil
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
	}
	for _, vdc := range e.Vdcs {
		if host, err := vdc.NextAvailableNode(); err == nil {
			return host, err
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
