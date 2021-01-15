#!/bin/sh

#kubectl apply -f ./volume.yaml
docker -- push gcr.io/ruirui-synerex-simulation/agent-provider:latest
docker -- push gcr.io/ruirui-synerex-simulation/master-provider:latest
docker -- push gcr.io/ruirui-synerex-simulation/visualization-provider:latest
docker -- push gcr.io/ruirui-synerex-simulation/gateway-provider:latest
docker -- push gcr.io/ruirui-synerex-simulation/worker-provider:latest
docker -- push gcr.io/ruirui-synerex-simulation/simulator:latest
docker -- push gcr.io/ruirui-synerex-simulation/synerex-server:latest
docker -- push gcr.io/ruirui-synerex-simulation/synerex-nodeserv:latest