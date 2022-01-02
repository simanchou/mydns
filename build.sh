#!/bin/bash

# replace your docker id before build

function BuildMaster() {
    sudo docker build . -f Dockerfile.master -t mydns-master
    sudo docker tag mydns-master simanchou/mydns-master
    sudo docker push simanchou/mydns-master
}

function BuildSlave() {
    sudo docker build . -f Dockerfile.slave -t mydns-slave
    sudo docker tag mydns-slave simanchou/mydns-slave
    sudo docker push simanchou/mydns-slave
}

function GenMasterApiDoc() {
    cd master
    goctl api plugin -plugin goctl-swagger="swagger -filename mydns.json -host 127.0.0.1 -basepath /api" -api master.api -dir .
}

function StartMasterApiDoc() {
    cd master
    sudo docker rm -f doc-mydns;sudo docker run -d --name doc-mydns -p 7000:8080 -e SWAGGER_JSON=/mydns/mydns.json -v $PWD:/mydns swaggerapi/swagger-ui
}

case $1 in
master)
  BuildMaster
  ;;
slave)
  BuildSlave
  ;;
doc-gen)
  GenMasterApiDoc
  ;;
doc-start)
  StartMasterApiDoc
  ;;
*)
  echo "unknown action"
  exit 1
  ;;
esac
