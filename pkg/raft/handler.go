package raft

import (
	"bytes"
	"errors"
	"net"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/detailyang/go-raft/pkg/raft/raftpb"
	"github.com/detailyang/go-raft/pkg/raft/transport"
)

func (r *Raft) runHandler() {
	r.logger.Infof("Run raft handler in background")

	err := r.transport.Accept()
	if err != nil {
		r.logger.Fatalln(err)
	}
}

func (r *Raft) handleRequest(conn net.Conn, request *transport.Payload) error {
	var err error
	defer func() {
		if err != nil {
			r.logger.Errorf("Failed to handle request(%s): %v", raftpb.MessageType(request.Type), err)
		}
	}()

	switch raftpb.MessageType(request.Type) {
	case raftpb.MessageType_Append_Entries_Request:
		appendEntriesRequest := &raftpb.AppendEntriesRequest{}
		err = proto.Unmarshal(request.Data, appendEntriesRequest)
		if err != nil {
			return err
		}

		return r.handleAppendEntriesRequest(conn, appendEntriesRequest)
	case raftpb.MessageType_Vote_Request:
		voteRequest := &raftpb.VoteRequest{}
		err = proto.Unmarshal(request.Data, voteRequest)
		if err != nil {
			return err
		}

		return r.handleVoteRequest(conn, voteRequest)
	case raftpb.MessageType_Install_Snapshot_Request:
		installRequest := &raftpb.InstallSnapshotRequest{}
		err = proto.Unmarshal(request.Data, installRequest)
		if err != nil {
			return err
		}

		return r.handleInstallSnapshotRequest(conn, installRequest)
	default:
		r.logger.Warnln("Receiving unknow request rpc: ", request.Type)
	}

	return nil
}

func (r *Raft) handleInstallSnapshotRequest(conn net.Conn, ir *raftpb.InstallSnapshotRequest) error {
	var (
		err     error
		n       int
		success = true
		req     = &RestoreRequest{
			index:   ir.LastLogIndex,
			term:    ir.LastLogTerm,
			replyCh: make(chan error, 1),
		}
		leader transport.Peer
	)

	if err := leader.Decode(ir.Leader); err != nil {
		return err
	}

	r.logger.Infof("Rceiving the installSnapshotRequest from %s", leader.String())

	// Receve snapshot if I'm a learner
	if !r.localState.isLearner {
		if ir.Term < r.localState.getTerm() {
			r.logger.Infof("Rceiving the older term, ignore it")
			return nil
		}
	}

	r.localState.setState(Follower)
	r.localState.setTerm(ir.Term)
	r.localState.setLeader(&leader)

	var (
		conf      Conf
		confIndex uint64
	)

	if err := conf.Decode(ir.Conf); err != nil {
		return err
	}

	confIndex = ir.ConfIndex

	si, err := r.snapshotStore.Create(ir.LastLogIndex, ir.LastLogTerm, conf, confIndex)
	if err != nil {
		r.logger.Errorf("Failed to create snapshot to install: %v", err)
		success = false
		goto SEND_RESPONSE
	}

	r.logger.Infof("Create snapshot at <%d:%d> with confIndex: %d", ir.LastLogIndex, ir.LastLogTerm, confIndex)

	n, err = si.Write(ir.Data)
	if err != nil {
		r.logger.Errorf("Failed to write snapshot: %v", err)
		si.Cancel()
		return err
	}

	if ir.Size != uint64(n) {
		si.Cancel()
		return errors.New("Failed to write total snapshot")
	}

	if err := si.Close(); err != nil {
		r.logger.Errorf("Failed to finalize snapshot: %v", err)
		goto SEND_RESPONSE
	}

	select {
	case r.restoreCh <- req:
	case <-time.After(r.options.SnapshotRestoreTimeout):
		return errors.New("restore snapshot timeout")
	case <-r.shutdownCh:
		return nil
	}
	// Wait until shutdown?
	select {
	case err := <-req.replyCh:
		if err != nil {
			r.logger.Errorf("Failed to restore snapshot: %v", err)
			goto SEND_RESPONSE
		}

	case <-r.shutdownCh:
		return nil
	}

	r.localState.setLastApplied(ir.LastLogIndex)
	r.localState.setLastSnapshot(ir.LastLogIndex, ir.LastLogTerm)

	r.confStore.latest = conf
	r.confStore.latestIndex = confIndex
	r.confStore.commited = conf
	r.confStore.commitedIndex = confIndex

SEND_RESPONSE:
	r.localState.setConcactAt()
	payload, err := newPayloadFromProto(uint8(raftpb.MessageType_Install_Snapshot_Response),
		&raftpb.InstallSnapshotResponse{
			Term:    r.GetTerm(),
			Success: success,
		})

	if err != nil {
		return err
	}

	_, err = conn.Write(payload.Encode())
	return err
}

func (r *Raft) handleVoteRequest(conn net.Conn, voteRequest *raftpb.VoteRequest) error {
	var candidater transport.Peer
	candidater.Decode(voteRequest.Candidate)

	r.logger.Infof("Receiving vote request from %s remote term:%d local term:%d",
		candidater.String(), voteRequest.Term, r.localState.getTerm())

	if leader := r.localState.getLeader(); leader != nil &&
		!leader.Equal(transport.DummyPeer) &&
		!leader.Equal(&candidater) {
		r.logger.Infof("Rejecting vote request from %s since we have a leader: %s",
			candidater.String(), leader)
		return nil
	}

	granted := true

	// Ignore an older term
	if voteRequest.Term < r.localState.getTerm() {
		r.logger.Infof("Rceiving the older term, ignore it")
		return nil
	}

	if voteRequest.Term > r.GetTerm() {
		r.localState.setState(Follower)
		r.localState.setTerm(voteRequest.Term)
	}

	// Check if we have voted yet
	lastVoteTerm, err := r.stableStore.GetUint64(keyLastVoteTerm)
	if err != nil && err != leveldb.ErrNotFound {
		r.logger.Errorf("Failed to get last vote term: %v", err)
		return err
	}

	lastVoteCandBytes, err := r.stableStore.Get(keyLastVoteCand)
	if err != nil && err != leveldb.ErrNotFound {
		r.logger.Infof("Failed to get last vote candidate: %v", err)
		return err
	}

	if lastVoteTerm == voteRequest.Term && lastVoteCandBytes != nil {
		r.logger.Warnf("Duplicate RequestVote for same term: %d", voteRequest.Term)
		if bytes.Compare(lastVoteCandBytes, voteRequest.Candidate) == 0 {
			// if time.Now().Sub(r.localState.getConcactAt()) > r.options.ElectionTimeout {
			granted = true
			// }
		} else {
			granted = false
		}
	}

	// Reject if we are more newer
	lastIndex, lastTerm := r.localState.getLastLog()
	if lastTerm > voteRequest.LastLogTerm {
		r.logger.Infof("Rejecting vote request from %v since our last term is newer (%d, %d)",
			candidater, lastTerm, voteRequest.LastLogTerm)
		granted = false
	}

	if lastTerm == voteRequest.LastLogTerm && lastIndex > voteRequest.LastLogIndex {
		r.logger.Infof("Rejecting vote request from %v since our last index is newer (%d, %d)",
			candidater, lastIndex, voteRequest.LastLogIndex)
		granted = false
	}

	r.localState.setConcactAt()

	// Persist a vote for safety
	if err := r.persistVote(voteRequest.Term, voteRequest.Candidate); err != nil {
		r.logger.Errorf("Failed to persist vote: %v", err)
		return err
	}

	payload, err := newPayloadFromProto(uint8(raftpb.MessageType_Vote_Response), &raftpb.VoteResponse{
		Term:    r.GetTerm(),
		Granted: granted,
	})

	if err != nil {
		return err
	}

	_, err = conn.Write(payload.Encode())
	return err
}

func (r *Raft) handleAppendEntriesRequest(conn net.Conn, appendEntriesRequest *raftpb.AppendEntriesRequest) error {
	var (
		err                   error
		leader                transport.Peer
		appendEntriesResponse = &raftpb.AppendEntriesResponse{
			Term:    r.localState.getTerm(),
			Success: false,
		}
	)

	if err = leader.Decode(appendEntriesRequest.Leader); err != nil {
		return err
	}

	r.logger.Infof("Receiving appendEntries request from %s remote term:%d local term:%d",
		leader.String(), appendEntriesRequest.Term, r.localState.getTerm())

	// Ignore an older term
	if appendEntriesRequest.Term < r.GetTerm() {
		goto SEND_RESPONSE
	}

	if appendEntriesRequest.Term > r.GetTerm() || r.GetState() != Follower {
		r.localState.setState(Follower)
		r.localState.setTerm(appendEntriesRequest.Term)
		appendEntriesResponse.Term = appendEntriesRequest.Term
	}

	r.localState.setLeader(&leader)

	// Check the prev log entry, let leader know where we are
	if appendEntriesRequest.PrevLogTerm > 0 {
		log, err := r.logStore.Get(appendEntriesRequest.PrevLogIndex)
		if err != nil {
			r.logger.Infof("Failed to get previous log: %d %v",
				appendEntriesRequest.PrevLogIndex, err)
			// We lack the log, so let the leader back off
			goto SEND_RESPONSE
		}

		if appendEntriesRequest.PrevLogTerm != log.Term {
			r.logger.Infof("Previous log term mis-match: ours: %d remote: %d",
				log.Term, appendEntriesRequest.PrevLogTerm)
			appendEntriesResponse.NoRetryBackoff = true
			goto SEND_RESPONSE
		}
	}

	// Check whether is heartbeat
	if len(appendEntriesRequest.Entries) == 0 {
		appendEntriesResponse.Success = true
		goto SEND_RESPONSE
	}

	if len(appendEntriesRequest.Entries) > 0 {
		lastLogIndex := r.GetLastLogIndex()
		newEntries := []*raftpb.Entry{}

		for i, entry := range appendEntriesRequest.Entries {
			if entry.Index > lastLogIndex {
				newEntries = appendEntriesRequest.Entries[i:]
				break
			}

			log, err := r.logStore.Get(entry.Index)
			if err != nil {
				r.logger.Infof("Failed to get log entry %d: %v",
					entry.Index, err)
				return err
			}

			if entry.Term != log.Term {
				r.logger.Infof("Clearing log suffix from %d to %d", entry.Index, lastLogIndex)
				if err = r.logStore.DeleteRange(entry.Index, lastLogIndex); err != nil {
					r.logger.Infof("[ERR] raft: Failed to clear log suffix: %v", err)
					return err
				}
			}
		}

		if n := len(newEntries); n > 0 {
			logs := make([]*Log, n)
			for i := 0; i < n; i++ {
				logs[i] = &Log{
					Term:  newEntries[i].Term,
					Index: newEntries[i].Index,
					Type:  LogType(newEntries[i].Type),
					Data:  newEntries[i].Data,
				}
			}

			if err := r.logStore.SetBatch(logs); err != nil {
				r.logger.Infof("[ERR] raft: Failed to append to logs: %v", err)
				return err
			}

			lastLog := newEntries[n-1]
			r.localState.setLastLog(lastLog.Index, lastLog.Term)
		}
	}

	appendEntriesResponse.Success = true

SEND_RESPONSE:

	r.localState.setConcactAt()

	appendEntriesResponse.LastLogIndex, appendEntriesResponse.LastLogTerm = r.localState.getLastLog()

	// Update the commit index
	if appendEntriesRequest.LeaderCommitIndex > 0 &&
		appendEntriesRequest.LeaderCommitIndex > r.localState.getCommitIndex() {
		index := min(appendEntriesRequest.LeaderCommitIndex, r.localState.getLastLogIndex())
		r.logStore.SetCommitIndex(index)
		r.localState.setCommitIndex(index)
		r.applyLogs(index)
	}

	payload, err := newPayloadFromProto(uint8(raftpb.MessageType_Append_Entries_Response), appendEntriesResponse)
	if err != nil {
		return err
	}

	_, err = conn.Write(payload.Encode())
	return err
}
