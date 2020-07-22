#!/bin/sh

echo "Please select"
echo "1. for uclab"
echo "2. for local"
read env

files="./pod-test/*.yaml"

if [ $env = '2' ] ; then
    files="./util/*.yaml"
fi

array=($files)

i=0
for filepath in $files; do
  echo $i: $filepath
  let i++
done

echo "Please select target"
read ti

echo file is ${array[ti]}
kubectl delete -f ${array[ti]}
