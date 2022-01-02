#!/bin/bash

# work dir
WORKING_DIR=$(cd `dirname $0`; pwd)
cd $WORKING_DIR

SERVICE_NAME=$(pwd|awk -F/ '{print $NF}')

function doc() {
    # generate
    goctl api plugin -plugin goctl-swagger="swagger -filename ${SERVICE_NAME}.json -host 127.0.0.2 -basepath /api" -api ${SERVICE_NAME}.api -dir .

    #     "description": "ops platform \n \nreturn of all master was wrapped by below struct: \n{ \n&nbsp;&nbsp;&nbsp;&nbsp;\"code\": int,         // only 2 options: 0 successful, -1 false \n&nbsp;&nbsp;&nbsp;&nbsp;\"msg\": \"error msg\",  // error message, it's null if successful \n&nbsp;&nbsp;&nbsp;&nbsp;\"data\": {object}     // replace master's responses here, it's null if error found \n} \n",

    # start doc service
    sudo docker rm -f doc-${SERVICE_NAME};sudo docker run -d --name doc-${SERVICE_NAME} -p 7000:8080 -e SWAGGER_JSON=/${SERVICE_NAME}/${SERVICE_NAME}.json -v $PWD:/${SERVICE_NAME} swaggerapi/swagger-ui
}

function code() {
    goctl api go -api ${SERVICE_NAME}.api -dir .
}

function run() {
    go run ${SERVICE_NAME}.go -f etc/${SERVICE_NAME}-api.yaml
}


ACTION=$1
case $ACTION in
doc)
  doc
  ;;
code)
  code
  ;;
run)
  run
  ;;
*)
  echo "unsupported action"
  ;;
esac
