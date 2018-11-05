package main

import (
	"errors"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/urfave/cli"

	"github.com/detailyang/go-raft/pkg/kvstore"
	"github.com/detailyang/go-raft/pkg/logger"
	"github.com/detailyang/go-raft/pkg/raft"
	"github.com/detailyang/go-raft/pkg/raft/store"
	"github.com/detailyang/go-raft/pkg/raft/transport"
)

// imports as package "cli"

var Version = "0.0.1"

func main() {
	app := cli.NewApp()
	app.Name = "raftd"
	app.Version = Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "conf",
			Usage: "specify config file path",
		},
	}
	app.Action = func(c *cli.Context) error {
		return start(c)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func start(c *cli.Context) error {

	path := c.String("conf")
	conf, err := loadConf(path)
	if err != nil {
		return err
	}

	go func() {
		log.Println(http.ListenAndServe(conf.Profile.Bind, nil))
	}()

	l := logger.NewLogger(conf.LogLevel)

	kvstore, err := kvstore.NewKVStore(conf.KVInfo.Bind, conf.KVInfo.Path)
	if err != nil {
		return err
	}

	Peers := make([]*transport.Peer, len(conf.RaftInfo.Nodes))
	for i, node := range conf.RaftInfo.Nodes {
		if len(node) != 2 {
			return errors.New("raft.node should be like [id, bind]")
		}

		Peers[i] = transport.NewPeer(
			transport.Voter,
			node[0],
			node[1],
		)
	}

	snapshotStore, err := store.NewSnapshotStore(conf.RaftInfo.SnapshotStorePath)
	if err != nil {
		return err
	}

	logStore, err := store.NewLogStore(conf.RaftInfo.LogStorePath)
	if err != nil {
		return err
	}

	stableStore, err := store.NewStableStore(conf.RaftInfo.StableStorePath)
	if err != nil {
		return err
	}

	shutdownCh := make(chan struct{})

	ro := raft.RaftOptions{
		ID:                     conf.RaftInfo.ID,
		Bind:                   conf.RaftInfo.Bind,
		Peers:                  Peers,
		Transport:              transport.NewTransport(conf.RaftInfo.Bind, 16, shutdownCh),
		IsLearner:              conf.RaftInfo.IsLearner,
		HeartbeatTimeout:       time.Duration(conf.RaftInfo.HeartbeatTimeout) * time.Millisecond,
		ElectionTimeout:        time.Duration(conf.RaftInfo.ElectionTimeout) * time.Millisecond,
		CommitTimeout:          time.Duration(conf.RaftInfo.CommitTimeout) * time.Millisecond,
		ReplicateTimeout:       time.Duration(conf.RaftInfo.ReplicateTimeout) * time.Millisecond,
		LeaderLeaseTimeout:     time.Duration(conf.RaftInfo.LeaderLeaseTimeout) * time.Millisecond,
		SnapshotStoreTimeout:   time.Duration(conf.RaftInfo.SnapshotStoreTimeout) * time.Millisecond,
		SnapshotRestoreTimeout: time.Duration(conf.RaftInfo.SnapshotRestoreTimeout) * time.Millisecond,
		SnapshotInterval:       time.Duration(conf.RaftInfo.SnapshotInterval) * time.Millisecond,
		SnapshotThreshold:      conf.RaftInfo.snapshotThreshold,
		Logger:                 l,
		LogStore:               logStore,
		StableStore:            stableStore,
		SnapshotStore:          snapshotStore,
		FSMStore:               kvstore,
		MaxAppendLogs:          conf.RaftInfo.MaxAppendLogs,
		ShutdownCh:             shutdownCh,
	}

	// raft.RaftOptions{}
	rf, err := raft.NewRaft(ro)
	if err != nil {
		return err
	}
	rf.Run()

	kvstore.SetRaft(rf)

	return kvstore.ListenAndServe()
}
