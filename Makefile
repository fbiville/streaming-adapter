PHONY: build generate-mocks test all

all: streaming/riff.pb.go build test

streaming/riff.pb.go: riff.proto
	protoc riff.proto --go_out=plugins=grpc:streaming

generate-mocks: streaming/riff.pb.go
	GO111MODULE=on mockery -name Riff_InvokeClient -dir streaming

build: streaming/riff.pb.go generate-mocks
	go build ./...

test:
	go test ./...
