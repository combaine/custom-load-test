#!/bin/bash -x
protoc --go_out=plugins=grpc:payload/ rpc.proto
python3 -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. rpc.proto
GO111MODULE=on go generate cmd/charge/main.go

pushd custom
for f in *.py; do
    python3 /usr/local/bin/cythonize -3 -i $f
done
popd
