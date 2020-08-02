#!/bin/sh

function ConfirmExecution() {

  echo "Please input version"
  echo "ex. latest"
  read version

  echo "----------------------------"
  echo "1. frontend"
  echo "2. backend"
  echo "3. map4-engine-api"
  echo "4. potree-api"
  echo "5. potree"
  echo "6. all"
  echo "----------------------------"
  echo "Please select build targets"
  echo "ex. 1 2 5 is frontend, backend, proxy"
  declare -a inputs=()
  read inputs

  echo $inputs

  for input in ${inputs[@]}; do
    if [ $input = '1' ] ; then
        echo "building frontend..."
        docker build -t map4_engine_web/frontend:${version} -f frontend/Dockerfile ./frontend

    elif [ $input = '2' ] ; then
        echo "building backend..."
        docker build -t map4_engine_web/backend:${version} -f backend/Dockerfile .

    elif [ $input = '3' ] ; then
        echo "building map4-engine-api..."
        docker build -t map4_engine_web/map4_engine_api:${version} -f map4_engine_api/Dockerfile ./map4_engine_api

    elif [ $input = '4' ] ; then
        echo "building potree-api..."
        docker build -t map4_engine_web/potree_api:${version} -f potree_api/Dockerfile ./potree_api

    elif [ $input = '5' ] ; then
        echo "building potree..."
        docker build -t map4_engine_web/potree:${version} -f potree/scripts/Dockerfile ./potree


    elif [ $input = '6' ] ; then
        echo "building all"
        docker build -t map4_engine_web/potree:${version} -f potree/scripts/Dockerfile ./potree
        docker build -t map4_engine_web/frontend:${version} -f frontend/Dockerfile ./frontend
        docker build -t map4_engine_web/backend:${version} -f backend/Dockerfile .
        docker build -t map4_engine_web/potree_api:${version} -f potree_api/Dockerfile ./potree_api
        docker build -t map4_engine_web/map4_engine_api:${version} -f map4_engine_api/Dockerfile ./map4_engine_api
    else
        echo "unknown number ${input}"

    fi
  done

}

ConfirmExecution

echo "----------------------------"
echo "finished!"
