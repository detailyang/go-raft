package kvstore

import (
	"time"

	"github.com/detailyang/go-raft/pkg/raft"
	"github.com/detailyang/go-raft/pkg/raft/transport"
	"github.com/tidwall/redcon"
)

func (kv *KVStore) handleClusterList(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 1 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}

	Peers := kv.raft.GetPeers()
	conn.WriteArray(len(Peers))
	for peer, _ := range Peers {
		conn.WriteBulkString(peer)
	}

	return
}

func (kv *KVStore) handleClusterJoin(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}

	id := string(cmd.Args[1])
	addr := string(cmd.Args[2])

	conf := kv.raft.GetConf()
	conf.AddPeer(&transport.Peer{
		Type: transport.Learner,
		ID:   id,
		Addr: addr,
	})

	if err := kv.raft.PropsalLog(conf.Encode(), raft.LogConfChange, 5*time.Second); err != nil {
		conn.WriteError(err.Error())
		return
	}

	conn.WriteString("OK")
	return
}

func (kv *KVStore) handleClusterLeft(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}

	id := string(cmd.Args[1])
	addr := string(cmd.Args[2])

	conf := kv.raft.GetConf()
	conf.DelPeer(&transport.Peer{
		Type: transport.Voter,
		ID:   id,
		Addr: addr,
	})

	if err := kv.raft.PropsalLog(conf.Encode(), raft.LogConfChange, 5*time.Second); err != nil {
		conn.WriteError(err.Error())
		return
	}

	conn.WriteString("OK")
	return
}

func (kv *KVStore) handleClusterLearnerLeft(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 3 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}

	id := string(cmd.Args[1])
	addr := string(cmd.Args[2])

	conf := kv.raft.GetConf()
	conf.DelPeer(&transport.Peer{
		Type: transport.Learner,
		ID:   id,
		Addr: addr,
	})

	if err := kv.raft.PropsalLog(conf.Encode(), raft.LogConfChange, 5*time.Second); err != nil {
		conn.WriteError(err.Error())
		return
	}

	conn.WriteString("OK")
	return
}
