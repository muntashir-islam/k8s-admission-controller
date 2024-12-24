import os
import logging
import json
from flask import Flask, request, jsonify

app = Flask(__name__)

# Configure logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)

DEPLOYMENT_RESOURCE = {"group": "apps", "version": "v1", "resource": "deployments"}
PREFIX = "stage-"


def generate_admission_response(uid, allowed, message=None, patches=None):
    """
    Generate a standardized admission response.
    """
    response = {"uid": uid, "allowed": allowed}
    if message:
        response["status"] = {"message": message}
    if patches:
        response["patchType"] = "JSONPatch"
        response["patch"] = patches
    return {"response": response}


@app.route("/mutate", methods=["POST"])
def mutate_deployment():
    """
    Adds the 'stage-' prefix to incoming Deployment names if missing.
    """
    logging.info("Received mutate request.")
    admission_review = request.get_json()

    uid = admission_review["request"]["uid"]
    resource = admission_review["request"]["resource"]
    deployment = admission_review["request"]["object"]

    if not all(resource.get(k) == v for k, v in DEPLOYMENT_RESOURCE.items()):
        message = f"Expected resource to be {DEPLOYMENT_RESOURCE}, but got {resource}"
        logging.error(message)
        return jsonify(generate_admission_response(uid, allowed=False, message=message))

    deployment_name = deployment["metadata"]["name"]
    logging.info(f"Original deployment name: {deployment_name}")

    if not deployment_name.startswith(PREFIX):
        new_name = f"{PREFIX}{deployment_name}"
        logging.info(f"Mutating deployment name to: {new_name}")

        patch = [{"op": "add", "path": "/metadata/name", "value": new_name}]
        return jsonify(
            generate_admission_response(uid, allowed=True, patches=json.dumps(patch))
        )

    logging.info("No mutation needed.")
    return jsonify(generate_admission_response(uid, allowed=True))


@app.route("/validate", methods=["POST"])
def validate_deployment():
    """
    Validates that Deployment names start with the 'stage-' prefix.
    """
    logging.info("Received validate request.")
    admission_review = request.get_json()

    uid = admission_review["request"]["uid"]
    resource = admission_review["request"]["resource"]
    deployment = admission_review["request"]["object"]

    if not all(resource.get(k) == v for k, v in DEPLOYMENT_RESOURCE.items()):
        message = f"Expected resource to be {DEPLOYMENT_RESOURCE}, but got {resource}"
        logging.error(message)
        return jsonify(generate_admission_response(uid, allowed=False, message=message))

    deployment_name = deployment["metadata"]["name"]
    logging.info(f"Validating deployment name: {deployment_name}")

    if not deployment_name.startswith(PREFIX):
        message = f"Deployment name '{deployment_name}' must start with '{PREFIX}'."
        logging.warning(message)
        return jsonify(generate_admission_response(uid, allowed=False, message=message))

    logging.info("Validation successful.")
    return jsonify(generate_admission_response(uid, allowed=True))


if __name__ == "__main__":
    # Load TLS certificate and key paths
    cert = os.environ.get("TLS_CERT", "/etc/certs/tls.crt")
    key = os.environ.get("TLS_KEY", "/etc/certs/tls.key")

    logging.info("Starting the webhook server...")
    app.run(host="0.0.0.0", port=8443, ssl_context=(cert, key))
