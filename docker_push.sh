#!/bin/sh

VERSION=1.0.0
echo "version is $VERSION"

echo "start"
docker push docker.pkg.github.com/ruihirano/flow_beta/synerex-nodeserv:$VERSION
docker push docker.pkg.github.com/ruihirano/flow_beta/synerex-server:$VERSION
docker push docker.pkg.github.com/ruihirano/flow_beta/master-provider:$VERSION
docker push docker.pkg.github.com/ruihirano/flow_beta/worker-provider:$VERSION
docker push docker.pkg.github.com/ruihirano/flow_beta/visualization-provider:$VERSION
docker push docker.pkg.github.com/ruihirano/flow_beta/agent-provider:$VERSION
docker push docker.pkg.github.com/ruihirano/flow_beta/gateway-provider:$VERSION
docker push docker.pkg.github.com/ruihirano/flow_beta/simulator:$VERSION

echo "----------------------------"
echo "finished!"