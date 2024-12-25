#!/bin/bash

echo "Removing certificates"
rm -rf certs
kubectl delete secrets webhook-certs
kubectl delete -f manifests/webhook-deployment.yaml
kubectl delete -f manifests/webhook-config.yaml