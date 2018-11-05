package raft

func (r *Raft) runFSM() {
	r.logger.Infoln("Run FSM handler in background")

	var (
		lastIndex uint64
		lastTerm  uint64
	)

	apply := func(applyLog *LogApply) error {
		if applyLog.Log.Type == LogCommand {
			if err := r.fsmStore.Apply(applyLog.Log); err != nil {
				return err
			}
		}

		lastIndex = applyLog.Log.Index
		lastTerm = applyLog.Log.Term
		r.localState.setLastApplied(applyLog.Log.Index)

		return nil
	}

	restore := func(req *RestoreRequest) error {
		meta, reader, err := r.snapshotStore.Open(req.term, req.index)
		if err != nil {
			return err
		}

		if err := r.fsmStore.Restore(reader); err != nil {
			return err
		}

		lastIndex = meta.Index
		lastTerm = meta.Term

		return nil
	}

	takeSnapshot := func(req *SnapshotRequest) error {
		snapshot, err := r.fsmStore.Snapshot()
		if err != nil {
			return err
		}

		req.snapshot = snapshot
		req.term = lastTerm
		req.index = lastIndex

		return nil
	}

	for {
		select {
		case applyLog := <-r.applyCh:
			applyLog.ReplyCh <- apply(applyLog)
		case req := <-r.snapshotCh:
			r.logger.Infof("Snapshot now")
			req.replyCh <- takeSnapshot(req)
		case req := <-r.restoreCh:
			req.replyCh <- restore(req)
		case <-r.shutdownCh:
			return
		}
	}
}
