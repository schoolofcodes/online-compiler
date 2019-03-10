#!/usr/bin/env bash

set -ex

parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "${parent_path}"
cd ..

cd app
go mod vendor
cd ..

docker build --tag=gcr.io/${PROJECT}/compiler .
docker push gcr.io/${PROJECT}/compiler

prev_ver=`gcloud container images list-tags gcr.io/${PROJECT}/compiler | grep -v latest | awk '{if(NR>1)print $1}'`
if [ ! -z "$prev_ver" ]; then
    yes | gcloud container images delete gcr.io/${PROJECT}/compiler@sha256:${prev_ver} --force-delete-tags
fi

gcloud compute instances stop online-compiler
gcloud compute instances start online-compiler