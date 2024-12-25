# Deloyment Validation Webhook

This repository contains a Golang-based Kubernetes Admission Controller that performs the following operations:

1. **Mutation Webhook**: Automatically adds the given prefix or `prod-` if no prefix given to the names of incoming Kubernetes `Deployments`.
2. **Validation Webhook**: Validates that Kubernetes `Deployments` have the given prefix or `prod-` if no prefix given in their names. If not, it rejects the deployment.


## Features

- Mutates deployment names by adding the given prefix  or `prod-` if it’s missing.
- Validates that deployment names start with given prefix or `prod-` if it’s missing to enforce naming conventions.
- Provides detailed logging for debugging and monitoring.
- Implements secure communication using TLS.

---

## Project Structure

```plaintext
.
├── main.go    # Main application code
├── Dockerfile                 # Dockerfile to containerize the app
├── README.md 
└── certs                      # Created after executing the create_k8s_object
    ├── tls.crt
    └── tls.key                # Project documentation
├── create_k8s_object.sh       # Generate TLS certs and deploy workload
├── delete_k8s_object.sh       # Delete TLS certs and deploy workload
└── manifests                  # Kubernetes manifests for deploying the webhook itself
    ├── weebhook-deployment.yaml
    └── webhook-config.yaml    #ValidatingWebhookConfiguration
```

---

## Prerequisites

- Golang 1.23.4
- Kubernetes cluster (v1.30+).
- `kubectl` CLI tool configured to access your cluster.
- TLS certificate and key for secure communication.

---

## Setup

### 1. Clone the Repository
```bash
git clone <repository-url>
cd <repository-folder>
```

### 2. Install Everything
```bash
./create_k8s_objects.sh
```

---

## Usage

### 1. Build and Deploy the Webhook

#### Build Docker Image
```bash
docker build -t deployment-validator:latest .

```

#### Deploy to Kubernetes
1. Push the image to your container registry (e.g., Docker Hub).
2. Update the image reference in `manifests/webhook-deployment.yaml`.
3. Apply the manifests:

```bash
kubectl apply -f manifests/
```

### 2. Test Deployment Validation

#### Create a Deployment with Valid Metadata
```bash
kubectl create deployment prod-app --image=nginx

```

#### Create a Deployment with Invalid Metadata
```bash
kubectl create deployment app
```
Expected output:
```
Error from server: admission webhook "validate.deployment-name.com" denied the request: Deployment name must start with 'stage-'
```

---

## Environment Variables

| Variable      | Description                     | Default     |
|---------------|---------------------------------|-------------|
| `TLS_CERT`    | Path to the TLS certificate     | `cert.pem`  |
| `TLS_KEY`     | Path to the TLS key             | `key.pem`   |

---

## Example ValidatingWebhookConfiguration

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: validate-deployment-prefix
webhooks:
  - name: validate.deployment.stage.com
    clientConfig:
      service:
        name: deployment-validator
        namespace: default
        path: "/validate"
      caBundle: ${ENCODED_CA}
    rules:
      - operations: ["CREATE"]
        apiGroups: ["apps"]
        apiVersions: ["v1"]
        resources: ["deployments"]
    admissionReviewVersions: ["v1"]
    failurePolicy: "Fail"
    sideEffects: "None"

---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: mutate-deployment-prefix
webhooks:
  - name: mutate.deployment.stage.com
    clientConfig:
      service:
        name: deployment-mutator
        namespace: default
        path: "/mutate"
      caBundle: ${ENCODED_CA}
    rules:
      - operations: ["CREATE"]
        apiGroups: ["apps"]
        apiVersions: ["v1"]
        resources: ["deployments"]
    admissionReviewVersions: ["v1"]
    failurePolicy: "Fail"
    sideEffects: "None"
```

---

## Contributing

1. Fork the repository.
2. Create a feature branch.
3. Submit a pull request with detailed explanations.

---

## License
This project is licensed under the MIT License. See the `LICENSE` file for details.

---
## Troubleshooting

### Common Errors

#### `x509: certificate relies on legacy Common Name field`
Ensure your certificate uses a Subject Alternative Name (SAN) matching `deployment-validator.default.svc`.

#### `expected response.uid="...", got ""`
Verify that the webhook responds with the correct `uid` field.

### Logs
Check the logs of the webhook pod for debugging:
```bash
kubectl logs <pod-name> -n default
```

