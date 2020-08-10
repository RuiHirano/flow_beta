#!/bin/sh

protoc -I . master.proto --go_out=plugins=grpc:.