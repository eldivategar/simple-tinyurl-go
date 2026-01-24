#!/bin/bash
protoc -I . -I third_party \
  --go_out . --go_opt paths=source_relative \
  --go-grpc_out . --go-grpc_opt paths=source_relative \
  --grpc-gateway_out . --grpc-gateway_opt paths=source_relative \
  proto/tinyurl/v1/tinyurl.proto
