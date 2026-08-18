#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

NAMESPACE="${NAMESPACE:-default}"
IMAGE_REPO_PREFIX="${IMAGE_REPO_PREFIX:-shubhindia/gpu-telemetry}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REPO_CSV_PATH="$ROOT_DIR/data/dcgm_metrics_20250718_134233.csv"
CSV_PATH="${CSV_PATH:-$REPO_CSV_PATH}"
CSV_FILE_NAME=""

print_step() {
  local message="$1"

  echo
  echo "============================================================"
  echo "$message"
  echo "============================================================"
}

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "missing required command: $1" >&2
		exit 1
	fi
}

install_chart() {
	local release="$1"
	local chart="$2"
	shift 2

	helm upgrade --install "$release" "$chart" \
		--namespace "$NAMESPACE" \
		--create-namespace \
		--wait \
		"$@"
}

wait_for_release_pods() {
	local release="$1"
	local name="$2"
	local selector="app.kubernetes.io/instance=${release},app.kubernetes.io/name=${name}"

	kubectl wait -n "$NAMESPACE" \
		--for=condition=Ready \
		pod \
		-l "$selector" \
		--timeout=300s
}

wait_for_workload() {
	local kind="$1"
	local name="$2"
	local release="$3"
	local app_name="$4"

	kubectl rollout status -n "$NAMESPACE" "$kind/$name" --timeout=300s
	wait_for_release_pods "$release" "$app_name"
}

require_command kubectl
require_command helm

if [[ ! -f "$CSV_PATH" ]]; then
	echo "csv file not found: $CSV_PATH" >&2
	echo "expected repo csv at: $REPO_CSV_PATH" >&2
	exit 1
fi

CSV_FILE_NAME="$(basename "$CSV_PATH")"

print_step "Preparing namespace and CSV volume..."
echo "Using CSV: $CSV_PATH"
kubectl get namespace "$NAMESPACE" >/dev/null 2>&1 || kubectl create namespace "$NAMESPACE"
kubectl apply -n "$NAMESPACE" -f "$ROOT_DIR/deploy/k8s/streamer-csv-pvc.yaml"
kubectl delete pod dcgm-metrics-loader -n "$NAMESPACE" --ignore-not-found >/dev/null 2>&1 || true
kubectl apply -n "$NAMESPACE" -f "$ROOT_DIR/deploy/k8s/streamer-csv-loader-pod.yaml"
kubectl wait -n "$NAMESPACE" --for=condition=Ready pod/dcgm-metrics-loader --timeout=120s
kubectl cp "$CSV_PATH" "$NAMESPACE/dcgm-metrics-loader:/mnt/csv/$CSV_FILE_NAME"

print_step "Installing Postgres..."
install_chart postgres "$ROOT_DIR/deploy/helm/postgres"
wait_for_workload deployment postgres-postgres postgres postgres

print_step "Installing queue..."
install_chart queue "$ROOT_DIR/deploy/helm/queue" \
  --set image.repository="$IMAGE_REPO_PREFIX-queue" \
  --set image.tag="$IMAGE_TAG"
wait_for_workload statefulset queue-queue queue queue

print_step "Installing streamer..."
install_chart streamer "$ROOT_DIR/deploy/helm/streamer" \
  -f "$ROOT_DIR/deploy/helm/streamer/values.pvc-example.yaml" \
  --set image.repository="$IMAGE_REPO_PREFIX-streamer" \
  --set image.tag="$IMAGE_TAG" \
  --set csv.fileName="$CSV_FILE_NAME"
wait_for_workload statefulset streamer-streamer streamer streamer

print_step "Installing processor..."
install_chart processor "$ROOT_DIR/deploy/helm/processor" \
  --set image.repository="$IMAGE_REPO_PREFIX-processor" \
  --set image.tag="$IMAGE_TAG"
wait_for_workload deployment processor-processor processor processor

print_step "Installing API..."
install_chart api "$ROOT_DIR/deploy/helm/api" \
  --set image.repository="$IMAGE_REPO_PREFIX-api" \
  --set image.tag="$IMAGE_TAG"
wait_for_workload deployment api-api api api

print_step "Installing monitoring..."
kubectl apply -n "$NAMESPACE" -f "$ROOT_DIR/deploy/k8s/monitoring/prometheus.yaml"
kubectl apply -n "$NAMESPACE" -f "$ROOT_DIR/deploy/k8s/monitoring/grafana.yaml"
kubectl rollout status -n "$NAMESPACE" deployment/prometheus --timeout=300s
kubectl rollout status -n "$NAMESPACE" deployment/grafana --timeout=300s

echo
echo "Bring-up complete."
echo
echo "Open these locally after port-forwarding:"
echo "  API Swagger:  http://127.0.0.1:8080/swagger"
echo "  API OpenAPI:  http://127.0.0.1:8080/openapi.json"
echo "  API GPUs:     http://127.0.0.1:8080/api/v1/gpus"
echo "  Grafana:      http://127.0.0.1:3000 (admin:admin)"
echo "  Prometheus:   http://127.0.0.1:9090"
echo
echo "Suggested port-forwards:"
echo "  kubectl port-forward -n $NAMESPACE svc/api-api 8080:8080"
echo "  kubectl port-forward -n $NAMESPACE svc/grafana 3000:3000"
echo "  kubectl port-forward -n $NAMESPACE svc/prometheus 9090:9090"
