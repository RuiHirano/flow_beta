#!/bin/sh

#kubectl apply -f ./volume.yaml
docker tag flow_beta/agent-provider:latest gcr.io/ruirui-synerex-simulation/agent-provider:latest
docker tag flow_beta/master-provider:latest gcr.io/ruirui-synerex-simulation/master-provider:latest
docker tag flow_beta/visualization-provider:latest gcr.io/ruirui-synerex-simulation/visualization-provider:latest
docker tag flow_beta/gateway-provider:latest gcr.io/ruirui-synerex-simulation/gateway-provider:latest
docker tag flow_beta/worker-provider:latest gcr.io/ruirui-synerex-simulation/worker-provider:latest
docker tag flow_beta/simulator:latest gcr.io/ruirui-synerex-simulation/simulator:latest
docker tag flow_beta/synerex-server:latest gcr.io/ruirui-synerex-simulation/synerex-server:latest
docker tag flow_beta/synerex-nodeserv:latest gcr.io/ruirui-synerex-simulation/synerex-nodeserv:latest