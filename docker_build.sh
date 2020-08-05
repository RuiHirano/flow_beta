#!/bin/sh

VERSION=1.0.0
echo "version is $VERSION"

echo "start build"
docker build -t docker.pkg.github.com/ruihirano/flow_beta/synerex-nodeserv:$VERSION -f server/synerex_nodeserv/Dockerfile ./server/synerex_nodeserv
docker build -t docker.pkg.github.com/ruihirano/flow_beta/synerex-server:$VERSION -f server/synerex_server/Dockerfile ./server/synerex_server
docker build -t docker.pkg.github.com/ruihirano/flow_beta/master-provider:$VERSION -f provider/master/Dockerfile .
docker build -t docker.pkg.github.com/ruihirano/flow_beta/worker-provider:$VERSION -f provider/worker/Dockerfile .
docker build -t docker.pkg.github.com/ruihirano/flow_beta/visualization-provider:$VERSION -f provider/visualization/Dockerfile .
docker build -t docker.pkg.github.com/ruihirano/flow_beta/agent-provider:$VERSION -f provider/agent/Dockerfile .
docker build -t docker.pkg.github.com/ruihirano/flow_beta/gateway-provider:$VERSION -f provider/gateway/Dockerfile .
docker build -t docker.pkg.github.com/ruihirano/flow_beta/simulator:$VERSION -f cli/Dockerfile ./cli

echo "----------------------------"
echo "finished!"