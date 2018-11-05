package main

import (
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Title    string   `toml:"title"`
	LogLevel string   `toml:"loglevel"`
	RaftInfo RaftInfo `toml:"raft"`
	KVInfo   KVInfo   `toml:"kvstore"`
	Profile  Profile  `toml:"profile"`
}

type Profile struct {
	Bind string `toml:"bind"`
}

type KVInfo struct {
	Bind string `toml:"bind"`
	Path string `toml:"path"`
}

type RaftInfo struct {
	ID        string `toml:"id"`
	Bind      string `toml:"bind"`
	IsLearner bool   `toml:"is_learner"`

	HeartbeatTimeout       int `toml:"heartbeat_timeout"`
	ElectionTimeout        int `toml:"election_timeout"`
	LeaderLeaseTimeout     int `toml:"leader_lease_timeout"`
	CommitTimeout          int `toml:"commit_timeout"`
	ReplicateTimeout       int `toml:"replicate_timeout"`
	SnapshotRestoreTimeout int `toml:"snapshot_restore_timeout"`
	SnapshotStoreTimeout   int `toml:"snapshot_store_timeout"`

	SnapshotInterval  int `toml:"snapshot_interval"`
	snapshotThreshold int `toml:"snapshot_threshold"`

	LogStorePath      string `toml:"log_store_path"`
	StableStorePath   string `toml:"stable_store_path"`
	SnapshotStorePath string `toml:"snapshot_store_path"`

	MaxCommpactLogs int `toml:"max_commpact_logs"`
	MaxAppendLogs   int `toml:"max_append_logs"`

	Nodes [][]string `toml:"nodes"`
}

func loadConf(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf Config
	if _, err := toml.Decode(string(data), &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}
