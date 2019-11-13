package raftstore

import (
	"time"

	etcdraftpb "github.com/coreos/etcd/raft/raftpb"
	"github.com/deepfabric/beehive/metric"
	"github.com/deepfabric/beehive/pb/metapb"
	"github.com/deepfabric/beehive/pb/raftcmdpb"
	"github.com/deepfabric/beehive/pb/raftpb"
	"github.com/fagongzi/util/protoc"
)

func (pr *peerReplica) startApplyingSnapJob() {
	pr.ps.applySnapJobLock.Lock()
	err := pr.store.addApplyJob(pr.shardID, "doApplyingSnapshotJob", pr.doApplyingSnapshotJob, pr.ps.setApplySnapJob)
	if err != nil {
		logger.Fatalf("shard %d add apply snapshot task failed with %+v",
			pr.shardID,
			err)
	}
	pr.ps.applySnapJobLock.Unlock()
}

func (pr *peerReplica) startRegistrationJob() {
	delegate := &applyDelegate{
		store:            pr.store,
		ps:               pr.ps,
		peerID:           pr.peer.ID,
		shard:            pr.ps.shard,
		term:             pr.getCurrentTerm(),
		applyState:       pr.ps.applyState,
		appliedIndexTerm: pr.ps.appliedIndexTerm,
	}

	err := pr.store.addApplyJob(pr.shardID, "doRegistrationJob", func() error {
		return pr.doRegistrationJob(delegate)
	}, nil)

	if err != nil {
		logger.Fatalf("shard %d add registration job failed, errors:\n %+v",
			pr.ps.shard.ID,
			err)
	}
}

func (pr *peerReplica) startApplyCommittedEntriesJob(shardID uint64, term uint64, commitedEntries []etcdraftpb.Entry) error {
	err := pr.store.addApplyJob(pr.shardID, "doApplyCommittedEntries", func() error {
		return pr.doApplyCommittedEntries(shardID, term, commitedEntries)
	}, nil)
	return err
}

func (pr *peerReplica) startRaftLogGCJob(shardID, startIndex, endIndex uint64) error {
	err := pr.store.addApplyJob(shardID, "doRaftLogGC", func() error {
		return pr.doRaftLogGC(shardID, startIndex, endIndex)
	}, nil)

	return err
}

func (s *store) startDestroyJob(shardID uint64, peer metapb.Peer) error {
	err := s.addApplyJob(shardID, "doDestroy", func() error {
		return s.doDestroy(shardID, peer)
	}, nil)

	return err
}

func (pr *peerReplica) startProposeJob(c *cmd, isConfChange bool) error {
	err := pr.store.addApplyJob(pr.shardID, "doPropose", func() error {
		return pr.doPropose(c, isConfChange)
	}, nil)

	return err
}

func (pr *peerReplica) startSplitCheckJob() error {
	shard := pr.ps.shard
	epoch := shard.Epoch
	startKey := encStartKey(&shard)
	endKey := encEndKey(&shard)

	err := pr.store.addSplitJob(func() error {
		return pr.doSplitCheck(epoch, startKey, endKey)
	})

	return err
}

func (ps *peerStorage) cancelApplyingSnapJob() bool {
	ps.applySnapJobLock.RLock()
	if ps.applySnapJob == nil {
		ps.applySnapJobLock.RUnlock()
		return true
	}

	ps.applySnapJob.Cancel()

	if ps.applySnapJob.IsCancelled() {
		ps.applySnapJobLock.RUnlock()
		return true
	}

	succ := !ps.isApplyingSnapshot()
	ps.applySnapJobLock.RUnlock()
	return succ
}

func (ps *peerStorage) resetApplyingSnapJob() {
	ps.applySnapJobLock.Lock()
	ps.applySnapJob = nil
	ps.applySnapJobLock.Unlock()
}

func (ps *peerStorage) resetGenSnapJob() {
	ps.genSnapJob = nil
	ps.snapTriedCnt = 0
}

func (pr *peerReplica) doPropose(c *cmd, isConfChange bool) error {
	value, ok := pr.store.delegates.Load(pr.shardID)
	if !ok {
		c.respShardNotFound(pr.shardID)
		return nil
	}

	delegate := value.(*applyDelegate)
	if delegate.shard.ID != pr.shardID {
		logger.Fatal("BUG: delegate id not match")
	}

	if isConfChange {
		changeC := delegate.getPendingChangePeerCMD()
		if nil != changeC && changeC.req != nil && changeC.req.Header != nil {
			delegate.notifyStaleCMD(changeC)
		}
		delegate.setPendingChangePeerCMD(c)
	} else {
		delegate.appendPendingCmd(c)
	}

	return nil
}

func (ps *peerStorage) doGenerateSnapshotJob() error {
	start := time.Now()

	if ps.genSnapJob == nil {
		logger.Fatalf("shard %d generating snapshot job is nil", ps.shard.ID)
	}

	applyState, err := ps.loadApplyState()
	if err != nil {
		logger.Fatalf("shard %d load snapshot failure, errors:\n %+v",
			ps.shard.ID,
			err)
		return nil
	} else if nil == applyState {
		logger.Fatalf("shard %d could not load snapshot", ps.shard.ID)
		return nil
	}

	var term uint64
	if applyState.AppliedIndex == applyState.TruncatedState.Index {
		term = applyState.TruncatedState.Term
	} else {
		entry, err := ps.loadLogEntry(applyState.AppliedIndex)
		if err != nil {
			return nil
		}

		term = entry.Term
		releaseEntry(entry)
	}

	state, err := ps.loadLocalState(nil)
	if err != nil {
		return nil
	}

	if state.State != raftpb.PeerNormal {
		logger.Errorf("shard %d snap seems stale, skip", ps.shard.ID)
		return nil
	}

	msg := &raftpb.SnapshotMessage{}
	msg.Header = raftpb.SnapshotMessageHeader{
		Shard: state.Shard,
		Term:  term,
		Index: applyState.AppliedIndex,
	}

	snapshot := etcdraftpb.Snapshot{}
	snapshot.Metadata.Term = msg.Header.Term
	snapshot.Metadata.Index = msg.Header.Index

	confState := etcdraftpb.ConfState{}
	for _, peer := range ps.shard.Peers {
		confState.Nodes = append(confState.Nodes, peer.ID)
	}
	snapshot.Metadata.ConfState = confState

	if ps.store.snapshotManager.Register(msg, Creating) {
		defer ps.store.snapshotManager.Deregister(msg, Creating)

		err = ps.store.snapshotManager.Create(msg)
		if err != nil {
			logger.Errorf("shard %d create snapshot file failure, errors:\n %+v",
				ps.shard.ID,
				err)
			return nil
		}
	}

	snapshot.Data = protoc.MustMarshal(msg)
	ps.genSnapJob.SetResult(snapshot)

	metric.ObserveSnapshotBuildingDuration(start)
	logger.Infof("shard %d snapshot created, epoch=<%s> term=<%d> index=<%d> ",
		ps.shard.ID,
		msg.Header.Shard.Epoch.String(),
		msg.Header.Term,
		msg.Header.Index)
	return nil
}

func (pr *peerReplica) doSplitCheck(epoch metapb.ShardEpoch, startKey, endKey []byte) error {
	if !pr.isLeader() {
		return nil
	}

	var size uint64
	var splitKey []byte

	size, splitKey, err := pr.store.getDataStorage(pr.shardID).SplitCheck(startKey, endKey, pr.store.opts.shardCapacityBytes)
	if err != nil {
		logger.Errorf("shard %d failed to scan split key, errors:\n %+v",
			pr.shardID,
			err)
		return err
	}

	if len(splitKey) == 0 {
		pr.sizeDiffHint = size
		return nil
	}

	logger.Infof("shard %d try to split, size=<%d> splitKey=<%d>",
		pr.shardID,
		size,
		splitKey)

	current := pr.ps.shard
	if current.Epoch.ShardVer != epoch.ShardVer {
		logger.Infof("shard %d epoch changed, need re-check later, current=<%+v> split=<%+v>",
			pr.shardID,
			current.Epoch,
			epoch)
		return nil
	}

	newShardID, newPeerIDs, err := pr.store.pd.GetRPC().AskSplit(newResourceAdapter(current))
	if err != nil {
		logger.Errorf("shard %d ask split failed with %+v",
			pr.shardID,
			err)
		return err
	}

	return pr.onAdmin(&raftcmdpb.AdminRequest{
		CmdType: raftcmdpb.Split,
		Split: &raftcmdpb.SplitRequest{
			SplitKey:   splitKey,
			NewShardID: newShardID,
			NewPeerIDs: newPeerIDs,
		},
	})
}