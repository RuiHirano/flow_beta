#!/bin/sh

kubectl apply -f master.yaml
kubectl apply -f worker.yaml
kubectl apply -f visualization.yaml
kubectl apply -f agent.yaml
kubectl apply -f simulator.yaml

