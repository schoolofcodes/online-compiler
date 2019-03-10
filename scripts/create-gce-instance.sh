#!/usr/bin/env bash

set -ex

gcloud compute firewall-rules create default-allow-https-8080 \
    --allow tcp:8080 \
    --source-ranges 0.0.0.0/0 \
    --target-tags https-server \
    --description "Allow port 8080 access to https-server" || true

yes | gcloud compute instances delete online-compiler --zone=us-east1-b || true

gcloud beta compute --project=online-compiler-215303 instances create-with-container online-compiler \
--zone=us-east1-b --machine-type=f1-micro --subnet=default --address=${COMPUTE_IP} \
--network-tier=PREMIUM --metadata=google-logging-enabled=true --maintenance-policy=MIGRATE \
--service-account=768654789015-compute@developer.gserviceaccount.com \
--scopes=https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/trace.append \
--tags=https-server,default-allow-https-8080 --image=cos-stable-70-11021-99-0 --image-project=cos-cloud --boot-disk-size=10GB \
--boot-disk-type=pd-standard --boot-disk-device-name=online-compiler \
--container-image=gcr.io/online-compiler-215303/compiler --container-restart-policy=always \
--container-mount-host-path=mount-path=/var/run/docker.sock,host-path=/var/run/docker.sock \
--labels=container-vm=cos-stable-70-11021-99-0
