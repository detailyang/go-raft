package kvstore

import (
	"encoding/json"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/detailyang/go-raft/pkg/raft"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tidwall/redcon"
)

const (
	OP_SET = iota
	OP_GET
	OP_DEL
)

type KVStore struct {
	addr string
	path string

	db     *leveldb.DB
	server *redcon.Server
	raft   *raft.Raft

	sync.RWMutex
	m map[string]string
}

type KVCommand struct {
	OP    int    `json:"int"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

func NewKVStore(addr, path string) (*KVStore, error) {
	kv := &KVStore{
		m:    make(map[string]string),
		path: path,
	}
	kv.server = redcon.NewServer(addr, kv.handler, kv.accept, kv.closed)
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	kv.db = db

	return kv, nil
}

func (kv *KVStore) SetRaft(raft *raft.Raft) {
	kv.raft = raft
}

func (kv *KVStore) ListenAndServe() error {
	// fmt.Println("kv store is listening at ", kv.server.Addr().String())
	return kv.server.ListenAndServe()
}

func (kv *KVStore) Apply(l *raft.Log) error {
	var cmd KVCommand

	if err := json.Unmarshal(l.Data, &cmd); err != nil {
		return err
	}

	switch cmd.OP {
	case OP_SET:
		return kv.applySet(cmd.Key, cmd.Value)
	case OP_GET:
		// value, ok := kv.applyGet(cmd.Key)
		// if !ok {
		// 	return errors.New("not found")
		// }
	}

	return nil
}

func (kv *KVStore) handler(conn redcon.Conn, cmd redcon.Command) {
	switch strings.ToLower(string(cmd.Args[0])) {
	default:
		conn.WriteError("ERR unknown command '" + string(cmd.Args[0]) + "'")
	case "ping":
		conn.WriteString("PONG")
	case "quit":
		conn.WriteString("OK")
		conn.Close()
	case "info":
		kv.handleInfo(conn, cmd)
	case "set":
		kv.handleSet(conn, cmd)
	case "snapshot":
		kv.handleSnapshot(conn, cmd)
	case "get":
		kv.handleGet(conn, cmd)
	case "cluster-left":
		kv.handleClusterLeft(conn, cmd)
	case "cluster-learner-left":
		kv.handleClusterLearnerLeft(conn, cmd)
	case "cluster-join":
		kv.handleClusterJoin(conn, cmd)
	case "cluster-list":
		kv.handleClusterList(conn, cmd)
	case "del":
		if len(cmd.Args) != 2 {
			conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
			return
		}
		// mu.Lock()
		// _, ok := items[string(cmd.Args[1])]
		// delete(items, string(cmd.Args[1]))
		// mu.Unlock()
		// if !ok {
		// 	conn.WriteInt(0)
		// } else {
		conn.WriteInt(1)
		// }
	}
}

func (kv *KVStore) accept(conn redcon.Conn) bool {
	// use this function to accept or deny the connection.
	log.Printf("accept: %s", conn.RemoteAddr())
	return true
}

func (kv *KVStore) closed(conn redcon.Conn, err error) {
}

// Restore is used to restore an FSM from a snapshot. It is not called
// concurrently with any other command. The FSM must discard all previous
// state.
func (kv *KVStore) Restore(io.Reader) error {
	return nil
}
