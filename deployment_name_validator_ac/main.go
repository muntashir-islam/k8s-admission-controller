package main

import (
	"flag"
	"fmt"
	"io"
	admission "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/testapigroup/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	scheme       = runtime.NewScheme()
	codec        = serializer.NewCodecFactory(scheme)
	deserializer = codec.UniversalDeserializer()
	prefix       string
)

func init() {
	_ = corev1.AddToScheme(scheme)
	_ = admission.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	prefix = os.Getenv("DEPLOYMENT_PREFIX")
	if prefix == "" {
		prefix = "prod-"
		log.Println("Using default prefix: prod-")
	} else {
		log.Printf("Using custom prefix: %s", prefix)
	}
}

type admissionHandlerFunc func(admission.AdmissionReview) *admission.AdmissionResponse
type admissionHandler struct {
	v1 admissionHandlerFunc
}

func createAdmissionHandler(f admissionHandlerFunc) admissionHandler {
	return admissionHandler{
		v1: f,
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request, admit admissionHandler) {
	var body []byte
	if r.Body != nil {
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
	}

	if len(body) == 0 {
		log.Println("Request body is empty")
		http.Error(w, "Request body is empty", http.StatusBadRequest)
		return
	}

	// Verify content type
	if r.Header.Get("Content-Type") != "application/json" {
		log.Println("Invalid content type, expecting application/json")
		http.Error(w, "Invalid content type, expecting application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Decode AdmissionReview
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		log.Printf("Failed to decode AdmissionReview: %v", err)
		http.Error(w, "Invalid AdmissionReview", http.StatusBadRequest)
		return
	}

	requestedAdmissionReview, ok := obj.(*admission.AdmissionReview)
	if !ok || requestedAdmissionReview.Request == nil {
		log.Println("Invalid AdmissionReview object")
		http.Error(w, "Invalid AdmissionReview object", http.StatusBadRequest)
		return
	}

	// Create a response
	responseAdmissionReview := &admission.AdmissionReview{
		Response: admit.v1(*requestedAdmissionReview),
	}

	if responseAdmissionReview.Response == nil {
		log.Println("Failed to create AdmissionResponse")
		http.Error(w, "Failed to create AdmissionResponse", http.StatusInternalServerError)
		return
	}

	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	responseAdmissionReview.SetGroupVersionKind(*gvk)

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

// handleMutate handles mutation requests
func handleMutate(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, createAdmissionHandler(mutate))
}

// handleValidate handles validation requests
func handleValidate(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, createAdmissionHandler(validate))
}
func mutate(ar admission.AdmissionReview) *admission.AdmissionResponse {
	log.Println("Performing mutation")
	resource := metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	if ar.Request.Resource != resource {
		log.Printf("Unexpected resource: %v", ar.Request.Resource)
		return nil
	}

	raw := ar.Request.Object.Raw
	var deployment appsv1.Deployment
	if _, _, err := deserializer.Decode(raw, nil, &deployment); err != nil {
		log.Printf("Failed to decode deployment: %v", err)
		return &admission.AdmissionResponse{
			Result: &metav1.Status{Message: err.Error()},
		}
	}

	newName := fmt.Sprintf("%s%s", prefix, deployment.GetName())
	patch := fmt.Sprintf(`[{"op": "add", "path": "/metadata/name", "value": "%s"}]`, newName)
	pt := admission.PatchTypeJSONPatch
	return &admission.AdmissionResponse{
		Allowed:   true,
		PatchType: &pt,
		Patch:     []byte(patch),
	}
}

func validate(ar admission.AdmissionReview) *admission.AdmissionResponse {
	log.Println("Performing validation")
	resource := metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	if ar.Request.Resource != resource {
		log.Printf("Unexpected resource: %v", ar.Request.Resource)
		return nil
	}

	raw := ar.Request.Object.Raw
	var deployment appsv1.Deployment
	if _, _, err := deserializer.Decode(raw, nil, &deployment); err != nil {
		log.Printf("Failed to decode deployment: %v", err)
		return &admission.AdmissionResponse{
			Result: &metav1.Status{Message: err.Error()},
		}
	}

	if !strings.HasPrefix(deployment.GetName(), prefix) {
		return &admission.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Deployment name must start with %s", prefix),
			},
		}
	}

	return &admission.AdmissionResponse{Allowed: true}
}

func main() {
	var tlsCert, tlsKey string
	flag.StringVar(&tlsCert, "tlsCert", "/etc/certs/tls.crt", "Path to TLS certificate")
	flag.StringVar(&tlsKey, "tlsKey", "/etc/certs/tls.key", "Path to TLS key")
	flag.Parse()

	http.HandleFunc("/mutate", handleMutate)
	http.HandleFunc("/validate", handleValidate)

	log.Println("Starting webhook server on :8443")
	log.Fatal(http.ListenAndServeTLS(":8443", tlsCert, tlsKey, nil))
}
