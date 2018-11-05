package kvstore

import (
	"fmt"

	"github.com/tidwall/redcon"
)

func (kv *KVStore) handleInfo(conn redcon.Conn, cmd redcon.Command) {
	if len(cmd.Args) != 1 {
		conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
		return
	}

	appliedIndex := kv.raft.GetLastApplied()
	commitIndex := kv.raft.GetCommitIndex()
	lastIndex := kv.raft.GetLastLogIndex()
	term := kv.raft.GetTerm()
	peer := kv.raft.GetPeer()
	leader := kv.raft.GetLeader()
	state := kv.raft.GetState()
	snapshotIndex, snapshotTerm := kv.raft.GetLastSnapshot()
	conf := kv.raft.GetConf()

	conn.WriteArray(8)
	conn.WriteBulkString(fmt.Sprintf("self:%s<%s>[%d]", peer, state, term))
	conn.WriteBulkString(fmt.Sprintf("leader:%s", leader))
	conn.WriteBulkString(fmt.Sprintf("appliedIndex:%d", appliedIndex))
	conn.WriteBulkString(fmt.Sprintf("commitIndex:%d", commitIndex))
	conn.WriteBulkString(fmt.Sprintf("lastIndex:%d", lastIndex))
	conn.WriteBulkString(fmt.Sprintf("snapshotIndex-snapshotTerm:%d-%d", snapshotIndex, snapshotTerm))
	conn.WriteBulkString(fmt.Sprintf("conf: %v", conf))
	conn.WriteBulkString("")

	return
}
