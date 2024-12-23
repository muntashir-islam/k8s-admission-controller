# Namespace Validation Webhook

This project is a Kubernetes Validating Admission Webhook that ensures namespaces adhere to specific conventions during their creation. The webhook validates namespace names, required labels, and annotations based on configurable rules.

## Features

- Validates namespace naming conventions (DNS-1123 compliant).
- Ensures namespaces include required labels and annotations.
- Provides detailed error messages for validation failures.
- Implements secure communication using TLS certificates.

---

## Project Structure

```plaintext
.
├── admission_controller.py    # Main application code
├── Dockerfile                 # Dockerfile to containerize the app
├── requirements.txt           # Python dependencies
├── README.md 
└── certs                      # Created after executing the create_k8s_object
    ├── tls.crt
    └── tls.key                # Project documentation
├── create_k8s_object.sh       # Generate TLS certs and deploy workload
└── manifests                  # Kubernetes manifests for deploying the webhook itself
    ├── weebhook-deployment.yaml
    └── webhook-config.yaml    #ValidatingWebhookConfiguration
```

---

## Prerequisites

- Python 3.10+
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

### 2. Install Dependencies
```bash
pip install -r requirements.txt
```

### 3. Generate TLS Certificates
Generate a self-signed certificate for the webhook server:

```bash
openssl genrsa -out certs/tls.key 2048
openssl req -new -key certs/tls.key -out certs/tls.csr -subj "/CN=namespace-validator.default.svc"
openssl x509 -req -extfile <(printf "subjectAltName=DNS:namespace-validator.default.svc") -in certs/tls.csr -signkey certs/tls.key -out certs/tls.crt

```

Base64 encode the certificate for the Kubernetes configuration:
```bash
ENCODED_CA=$(cat certs/tls.crt | base64 | tr -d '\n')
sed -e 's@${ENCODED_CA}@'"$ENCODED_CA"'@g' <"manifests/webhook-config.yml" | kubectl create -f -
```

### 4. Update Kubernetes Manifest
Previous script will autometically replace the `<CA_BUNDLE>` placeholder in `manifests/webhook-config.yaml` with the contents of `cert.base64`:

```yaml
caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1...
```

---

## Usage

### 1. Build and Deploy the Webhook

#### Build Docker Image
```bash
docker build -t namespace-validator:latest .
```

#### Deploy to Kubernetes
1. Push the image to your container registry (e.g., Docker Hub).
2. Update the image reference in `manifests/webhook-deployment.yaml`.
3. Apply the manifests:

```bash
kubectl apply -f manifests/
```

### 2. Test Namespace Validation

#### Create a Namespace with Valid Metadata
```bash
kubectl create namespace valid-namespace \
  --dry-run=client -o yaml |
  kubectl annotate --local -f - "owner=test" "purpose=testing" |
  kubectl label --local -f - "team=dev" "environment=staging" |
  kubectl apply -f -
```

#### Create a Namespace with Invalid Metadata
```bash
kubectl create namespace invalid_namespace
```
Expected output:
```
Error from server: Namespace name 'invalid_namespace' does not follow naming convention.
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
  name: namespace-creation-validator
webhooks:
  - name: validate.namespace-creation.com
    clientConfig:
      service:
        name: namespace-validator
        namespace: default
        path: "/validate"
      caBundle: <CA_BUNDLE>
    rules:
      - operations: ["CREATE"]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["namespaces"]
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

## Acknowledgments
- Kubernetes Documentation
- Flask Framework Documentation

---

## Troubleshooting

### Common Errors

#### `x509: certificate relies on legacy Common Name field`
Ensure your certificate uses a Subject Alternative Name (SAN) matching `namespace-validator.default.svc`.

#### `expected response.uid="...", got ""`
Verify that the webhook responds with the correct `uid` field.

### Logs
Check the logs of the webhook pod for debugging:
```bash
kubectl logs <pod-name> -n default
```

