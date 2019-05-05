#!/bin/bash -x
protoc --go_out=plugins=grpc:payload/ rpc.proto
python3 -m grpc_tools.protoc -I.  --python_out=. --grpc_python_out=. rpc.proto
go generate cmd/charge/main.go
