package transport

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync/atomic"
)

var (
	DummyPeer = newDummyPeer()
)

const (
	Voter   = uint32(0)
	Learner = iota
)

type Peer struct {
	Type uint32
	ID   string
	Addr string
}

func init() {
	gob.Register(Peer{})
}

func newDummyPeer() *Peer {
	return &Peer{
		ID:   "dummy",
		Addr: "127.0.0.1:0",
	}
}

func NewPeer(t uint32, id, addr string) *Peer {
	return &Peer{
		Type: t,
		ID:   id,
		Addr: addr,
	}
}

func (p *Peer) Clone() Peer {
	return Peer{
		Type: p.Type,
		ID:   p.ID,
		Addr: p.Addr,
	}
}

func (p *Peer) Encode() []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(p)

	return buf.Bytes()
}

func (p *Peer) Decode(data []byte) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(p)
}

func (p *Peer) PartialEqual(t *Peer) bool {
	return p.ID == t.ID && p.Addr == t.Addr
}

func (p *Peer) Equal(t *Peer) bool {
	return p.ID == t.ID && p.Addr == t.Addr && p.Type == t.Type
}

func (p *Peer) LString() string {
	return fmt.Sprintf("%s<%s>", p.Addr, p.ID)
}

func (p *Peer) String() string {
	if p.GetType() == Voter {
		return fmt.Sprintf("%s<%s>[%s]", p.Addr, p.ID, "voter")
	}

	return fmt.Sprintf("%s<%s>[%s]", p.Addr, p.ID, "learner")
}

func (p *Peer) SetType(t uint32) {
	atomic.StoreUint32(&p.Type, t)
}

func (p *Peer) GetType() uint32 {
	return atomic.LoadUint32(&p.Type)
}
