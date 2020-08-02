#!/bin/sh

protoc -I . simulation.proto --go_out=plugins=grpc:.