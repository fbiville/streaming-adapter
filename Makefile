PHONY: build generate-mocks test all

all: riff.pb.go build test

riff.pb.go: riff.proto
	protoc riff.proto --go_out=plugins=grpc:streaming_proto

generate-mocks: riff.pb.go
	GO111MODULE=on mockery -name Riff_InvokeClient -dir streaming_proto

build: riff.pb.go generate-mocks
	go build ./...

test:
	go test ./...
