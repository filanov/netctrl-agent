.PHONY: build test clean run

build:
	go build -o bin/netctrl-agent cmd/agent/main.go

test:
	go test -v ./...

clean:
	rm -rf bin/

run:
	go run cmd/agent/main.go --cluster-id=test-cluster
