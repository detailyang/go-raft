all:
	GO111MODULE=on go build -o bin/raftd cmd/raftd/*.go
.PHONY: all

clean:
	@rm -rf bin/
