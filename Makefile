arg=11

node:
		kubectl get nodes -o wide

pod:
		kubectl get pods -o wide

svc:
		kubectl get svc -o wide

master:
		kubectl logs -f master -c master-provider

simulator:
		kubectl exec -it simulator -c simulator bash

worker:
		kubectl logs -f workertest -c worker-provider

database:
		kubectl logs -f database${arg} -c database-provider

agent:
		kubectl logs -f agent11 -c agent-provider

vis:
		kubectl logs -f visualization -c visualization-provider

gateway:
		kubectl logs -f gateway${arg} -c gateway-provider

apply:
		kubectl apply -f ./kube/resource2/sample.yaml

delete:
		kubectl delete -f ./kube/resource2/sample.yaml

app:
		kubectl apply -f ./kube/util/master-simulator.yaml

del:
		kubectl delete -f ./kube/util/master-simulator.yaml

gen-rsc:
		bash ./kube/util/gen-rsc.sh

build:
		bash ./docker_build.sh

push-lab:
		bash ./docker_push_uclab.sh

build-lab:
		bash ./docker_build_uclab.sh

desc:
		cd kube && bash kube_describe.sh

log:
		cd kube && bash kube_log.sh

exec:
		cd kube && bash kube_exec.sh

del2: 
		kubectl delete pod agent11 agent12 agent21 agent22 agent31 agent32 worker11 worker12 worker21 worker22 worker31 worker32 gateway1111 gateway1112 gateway1121 gateway1212 gateway1222 gateway2121 gateway2122 gateway2131 gateway2222 gateway2232 gateway3131 gateway3132