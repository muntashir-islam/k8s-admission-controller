#!/bin/bash

echo "Creating certificates"
mkdir certs
openssl genrsa -out certs/tls.key 2048
openssl req -new -key certs/tls.key -out certs/tls.csr -subj "/CN=webhook.default.svc"
openssl x509 -req -extfile <(printf "subjectAltName=DNS:webhook.default.svc") -in certs/tls.csr -signkey certs/tls.key -out certs/tls.crt

echo "Creating Webhook Server TLS Secret"
kubectl create secret tls webhook-certs \
    --cert "certs/tls.crt" \
    --key "certs/tls.key"

echo "Creating Webhook Server Deployment"
kubectl create -f manifests/webhook-deployment.yaml

echo "Creating K8s Webhooks for namespce"
ENCODED_CA=$(cat certs/tls.crt | base64 | tr -d '\n')
sed -e 's@${ENCODED_CA}@'"$ENCODED_CA"'@g' <"manifests/webhook-config.yaml" | kubectl create -f -