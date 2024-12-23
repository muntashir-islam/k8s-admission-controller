from flask import Flask, request, jsonify
import re
import os

app = Flask(__name__)

# Required labels and annotations
REQUIRED_LABELS = ["team", "environment"]
REQUIRED_ANNOTATIONS = ["owner", "purpose"]
NAMESPACE_REGEX = r"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"  # Kubernetes DNS-1123 naming convention

def generate_response(uid, allowed, message=None):
    """
    Generate a standard AdmissionReview response.
    """
    response = {
        "apiVersion": "admission.k8s.io/v1",
        "kind": "AdmissionReview",
        "response": {
            "uid": uid,
            "allowed": allowed
        }
    }
    if message:
        response["response"]["status"] = {"message": message}
    return jsonify(response)

@app.route("/validate", methods=["POST"])
def validate_namespace():
    # Parse the AdmissionReview request
    request_info = request.get_json()
    uid = request_info.get("request", {}).get("uid", "")
    namespace = request_info.get("request", {}).get("object", {})
    
    # Extract namespace metadata
    name = namespace.get("metadata", {}).get("name", "")
    labels = namespace.get("metadata", {}).get("labels", {})
    annotations = namespace.get("metadata", {}).get("annotations", {})

    # Validate namespace name
    if not re.match(NAMESPACE_REGEX, name):
        return generate_response(uid, False, f"Namespace name '{name}' does not follow naming convention.")

    # Validate required labels
    missing_labels = [label for label in REQUIRED_LABELS if label not in labels]
    if missing_labels:
        return generate_response(uid, False, f"Missing required labels: {', '.join(missing_labels)}")

    # Validate required annotations
    missing_annotations = [annotation for annotation in REQUIRED_ANNOTATIONS if annotation not in annotations]
    if missing_annotations:
        return generate_response(uid, False, f"Missing required annotations: {', '.join(missing_annotations)}")

    # If all checks pass
    return generate_response(uid, True)

if __name__ == "__main__":
    # Load SSL certificate and key
    cert = os.environ.get("TLS_CERT", "cert.pem")
    key = os.environ.get("TLS_KEY", "key.pem")
    app.run(host="0.0.0.0", port=8443, ssl_context=(cert, key))