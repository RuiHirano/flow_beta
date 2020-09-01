# flow_beta

## Getting Started

### 1. pull image
```
bash docker_pull.sh
```
image file is
- synerex-server
- synerex-nodeserv
- master-provider
- worker-provider
- visualization-provider
- agent-provider
- gateway-provider
- simulator

### 2. apply yaml

```
make apply
// or kubectl apply -f ./kube/resource2/sample.yaml
```

### 3. show master provider terminal
- in your new terminal
```
make master
```

### 4. send order from simulator

```
make simulator
```
- in simualtor pod
```
./simulator order set agent -n 3000
./simulator order start
./simulator order stop
```

### 5. view HarmowareVis-Monitor
You can watch monitor at http://localhost:30000


### 6. delete yaml
```
make delete
```

## Costomize Simulator
coming soon...