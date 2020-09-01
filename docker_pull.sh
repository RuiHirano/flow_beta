#!/bin/sh

VERSION=1.0.0
echo "version is $VERSION"

echo "start"
docker pull docker.pkg.github.com/ruihirano/flow_beta/synerex-nodeserv:$VERSION
docker pull docker.pkg.github.com/ruihirano/flow_beta/synerex-server:$VERSION
docker pull docker.pkg.github.com/ruihirano/flow_beta/master-provider:$VERSION
docker pull docker.pkg.github.com/ruihirano/flow_beta/worker-provider:$VERSION
docker pull docker.pkg.github.com/ruihirano/flow_beta/visualization-provider:$VERSION
docker pull docker.pkg.github.com/ruihirano/flow_beta/agent-provider:$VERSION
docker pull docker.pkg.github.com/ruihirano/flow_beta/gateway-provider:$VERSION
docker pull docker.pkg.github.com/ruihirano/flow_beta/simulator:$VERSION

echo "----------------------------"
echo "finished!"