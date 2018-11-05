# departed

Sadly, I lost about 5000 lines go code because of the stupid gitignore rule: 

 * patterns match files in any directory

always remember using`git status` to track the repo and commit the large number of buggly code in startup then reorganize the commits:)

# go-raft

It's my toy-level raft implement of [CONSENSUS: BRIDGING THEORY AND PRACTICE](https://ramcloud.stanford.edu/~ongaro/thesis.pdf) which no perf and no tests in time. And I just want to know how raft works:) Anyone want to use it in production, you are at at your own risk. It's better to use [etcd/raft](https://github.com/etcd-io/etcd/tree/master/raft) or [hashicorp/raft](https://github.com/hashicorp/raft) to embrace distributed system.


### Prerequisites

You need sure Go's version >= 1.10 as the following:

```
go version
```

### Building

One command to build

```
make
```

## Tests

No tests:(. So dont use it in production

## Getting Started

There is a simple kv store exposed with redis-protocol on specified port. So we can test functionan by `redis-cli`

### Two nodes

* startup nodes:
    1. bin/raftd --conf conf/raft0.toml
    2. bin/raftd --conf conf/raft1.toml
* set kv
    1. redis-cli -p 7000 set a b
* get kv
    1. redis-cli -p 7001 get a

### Two nodes and add new node
* startup nodes:
    1. bin/raftd --conf conf/raft0.toml
    2. bin/raftd --conf conf/raft1.toml
* startup new node:
    1. change conf/raft2.toml is_learner to false
    2. bin/raftd --conf conf/raft2.toml
* tell leader to add node:
    1. redis-cli -p 7000 cluster-add raft2 127.0.0.1:8002
* wait new node to be follower:
    1. change conf/raft2.toml is_learer to true else it will panic
    2. bin/raftd --conf conf/raft2.toml

### three nodes and delete node
* startup nodes:
    1. bin/raftd --conf conf/raft0.toml
    2. bin/raftd --conf conf/raft1.toml 
    3. bin/raftd --conf conf/raft2.toml 
* delete node:
    1. redis-cli -p 7000 cluster-left raft2 127.0.0.1:8002

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details