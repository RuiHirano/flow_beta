#!/bin/sh

echo "start"
docker image build -t synerex-simulation/nodeid-server:${version} -f server/synerex_nodeserv/Dockerfile .
docker image build -t synerex-simulation/synerex-server:${version} -f server/synerex_server/Dockerfile .
docker image build -t synerex-simulation/master-provider:${version} -f provider/master/Dockerfile .
docker image build -t synerex-simulation/worker-provider:${version} -f provider/worker/Dockerfile .
docker image build -t synerex-simulation/agent-provider:${version} -f provider/agent/Dockerfile .
docker image build -t synerex-simulation/visualization-provider:${version} -f provider/visualization/Dockerfile .
docker image build -t synerex-simulation/gateway-provider:${version} -f provider/gateway/Dockerfile .
docker image build -t synerex-simulation/simulator:${version} -f cli/Dockerfile .

echo "----------------------------"
echo "finished!"