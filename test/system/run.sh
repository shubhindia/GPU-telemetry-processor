#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-gpu-telemetry-system}"
NAMESPACE="${SYSTEM_TEST_NAMESPACE:-gpu-telemetry-system}"
CSV_PATH="${CSV_PATH:-$ROOT_DIR/data/dcgm_metrics_20250718_134233.csv}"
CSV_FILE_NAME="$(basename "$CSV_PATH")"
IMAGE_REPO_PREFIX="${IMAGE_REPO_PREFIX:-local/gpu-telemetry}"
IMAGE_TAG="${IMAGE_TAG:-system}"
SYSTEM_TEST_DOCKER="${SYSTEM_TEST_DOCKER:-docker}"
SYSTEM_TEST_CREATE_CLUSTER="${SYSTEM_TEST_CREATE_CLUSTER:-1}"
SYSTEM_TEST_KEEP_CLUSTER="${SYSTEM_TEST_KEEP_CLUSTER:-0}"
SYSTEM_TEST_KEEP_NAMESPACE="${SYSTEM_TEST_KEEP_NAMESPACE:-0}"

QUEUE_REPLICAS=3
STREAMER_REPLICAS=2

created_cluster=0

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

decode_base64() {
	if base64 --help 2>/dev/null | grep -q -- '-d'; then
		base64 -d
	else
		base64 -D
	fi
}

cluster_exists() {
	kind get clusters | grep -Fxq "$KIND_CLUSTER_NAME"
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
	local kind_name="$1"
	local resource_name="$2"
	local release="$3"
	local app_name="$4"

	kubectl rollout status -n "$NAMESPACE" "$kind_name/$resource_name" --timeout=300s
	wait_for_release_pods "$release" "$app_name"
}

cluster_request() {
	local url="$1"
	kubectl exec -n "$NAMESPACE" dcgm-metrics-loader -- wget -qO- "$url"
}

wait_for_json_query() {
	local url="$1"
	local jq_expr="$2"
	local description="$3"
	local timeout_seconds="${4:-300}"
	local start_seconds
	start_seconds="$(date +%s)"

	while true; do
		local body=""

		body="$(cluster_request "$url" 2>/dev/null || true)"
		if [[ -n "$body" ]] && jq -e "$jq_expr" >/dev/null 2>&1 <<<"$body"; then
			echo "$body"
			return 0
		fi

		if (($(date +%s) - start_seconds >= timeout_seconds)); then
			echo "timed out waiting for ${description}" >&2
			if [[ -n "$body" ]]; then
				echo "$body" >&2
			fi
			return 1
		fi

		sleep 5
	done
}

sum_queue_counter() {
	local counter="$1"
	local total=0
	local ordinal

	for ordinal in $(seq 0 $((QUEUE_REPLICAS - 1))); do
		local url="http://queue-queue-${ordinal}.queue-queue.${NAMESPACE}.svc.cluster.local:8080/stats"
		local body
		body="$(cluster_request "$url")"
		total=$((total + $(jq -r ".counters.${counter}" <<<"$body")))
	done

	echo "$total"
}

wait_for_queue_activity() {
	local counter="$1"
	local description="$2"
	local timeout_seconds="${3:-300}"
	local start_seconds
	start_seconds="$(date +%s)"

	while true; do
		local total
		total="$(sum_queue_counter "$counter")"
		if ((total > 0)); then
			echo "$total"
			return 0
		fi

		if (($(date +%s) - start_seconds >= timeout_seconds)); then
			echo "timed out waiting for ${description}" >&2
			return 1
		fi

		sleep 5
	done
}

postgres_exec() {
	local sql="$1"
	local pod_name
	local pg_user
	local pg_password
	local pg_database

	pod_name="$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/instance=postgres,app.kubernetes.io/name=postgres -o jsonpath='{.items[0].metadata.name}')"
	pg_user="$(kubectl get secret postgres-postgres-auth -n "$NAMESPACE" -o jsonpath='{.data.POSTGRES_USER}' | decode_base64)"
	pg_password="$(kubectl get secret postgres-postgres-auth -n "$NAMESPACE" -o jsonpath='{.data.POSTGRES_PASSWORD}' | decode_base64)"
	pg_database="$(kubectl get secret postgres-postgres-auth -n "$NAMESPACE" -o jsonpath='{.data.POSTGRES_DB}' | decode_base64)"

	kubectl exec -n "$NAMESPACE" "$pod_name" -- env PGPASSWORD="$pg_password" \
		psql -U "$pg_user" -d "$pg_database" -Atc "$sql"
}

dump_diagnostics() {
	echo
	echo "System test diagnostics"
	echo "-----------------------"
	kubectl get pods -n "$NAMESPACE" -o wide || true
	kubectl get pvc -n "$NAMESPACE" || true
	kubectl get events -n "$NAMESPACE" --sort-by=.metadata.creationTimestamp | tail -n 50 || true

	for selector in \
		'app.kubernetes.io/instance=queue,app.kubernetes.io/name=queue' \
		'app.kubernetes.io/instance=streamer,app.kubernetes.io/name=streamer' \
		'app.kubernetes.io/instance=processor,app.kubernetes.io/name=processor' \
		'app.kubernetes.io/instance=api,app.kubernetes.io/name=api' \
		'app.kubernetes.io/instance=postgres,app.kubernetes.io/name=postgres'; do
		local pod
		pod="$(kubectl get pods -n "$NAMESPACE" -l "$selector" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
		if [[ -n "$pod" ]]; then
			echo
			echo "Logs: $pod"
			kubectl logs -n "$NAMESPACE" "$pod" --tail=200 || true
		fi
	done
}

cleanup() {
	local exit_code="$1"

	if ((exit_code != 0)); then
		dump_diagnostics
	fi

	if [[ "$SYSTEM_TEST_KEEP_NAMESPACE" != "1" ]]; then
		kubectl delete namespace "$NAMESPACE" --ignore-not-found --wait=false >/dev/null 2>&1 || true
	fi

	if ((created_cluster == 1)) && [[ "$SYSTEM_TEST_KEEP_CLUSTER" != "1" ]]; then
		kind delete cluster --name "$KIND_CLUSTER_NAME" >/dev/null 2>&1 || true
	fi
}

trap 'cleanup "$?"' EXIT

require_command kind
require_command kubectl
require_command helm
require_command jq
require_command "$SYSTEM_TEST_DOCKER"

if [[ ! -f "$CSV_PATH" ]]; then
	echo "csv file not found: $CSV_PATH" >&2
	exit 1
fi

if cluster_exists; then
	kubectl config use-context "kind-$KIND_CLUSTER_NAME" >/dev/null
elif [[ "$SYSTEM_TEST_CREATE_CLUSTER" == "1" ]]; then
	print_step "Creating kind cluster..."
	kind create cluster --name "$KIND_CLUSTER_NAME" --wait 300s
	created_cluster=1
else
	echo "kind cluster not found: $KIND_CLUSTER_NAME" >&2
	exit 1
fi

kubectl cluster-info >/dev/null

print_step "Building local images..."
$(command -v make) -C "$ROOT_DIR" build-images \
	DOCKER="$SYSTEM_TEST_DOCKER" \
	IMAGE_REPO_PREFIX="$IMAGE_REPO_PREFIX" \
	IMAGE_TAG="$IMAGE_TAG"

print_step "Loading images into kind..."
kind load docker-image --name "$KIND_CLUSTER_NAME" \
	"${IMAGE_REPO_PREFIX}-queue:${IMAGE_TAG}" \
	"${IMAGE_REPO_PREFIX}-streamer:${IMAGE_TAG}" \
	"${IMAGE_REPO_PREFIX}-processor:${IMAGE_TAG}" \
	"${IMAGE_REPO_PREFIX}-api:${IMAGE_TAG}"

print_step "Preparing namespace and CSV volume..."
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n "$NAMESPACE" -f "$ROOT_DIR/deploy/k8s/streamer-csv-pvc.yaml"
kubectl delete pod dcgm-metrics-loader -n "$NAMESPACE" --ignore-not-found >/dev/null 2>&1 || true
kubectl apply -n "$NAMESPACE" -f "$ROOT_DIR/deploy/k8s/streamer-csv-loader-pod.yaml"
kubectl wait -n "$NAMESPACE" --for=condition=Ready pod/dcgm-metrics-loader --timeout=180s
kubectl exec -n "$NAMESPACE" dcgm-metrics-loader -- sh -c 'rm -f /mnt/csv/*'
kubectl cp "$CSV_PATH" "$NAMESPACE/dcgm-metrics-loader:/mnt/csv/$CSV_FILE_NAME"

print_step "Installing Postgres..."
install_chart postgres "$ROOT_DIR/deploy/helm/postgres"
wait_for_workload deployment postgres-postgres postgres postgres

print_step "Installing queue..."
install_chart queue "$ROOT_DIR/deploy/helm/queue" \
	--set replicaCount="$QUEUE_REPLICAS" \
	--set image.repository="$IMAGE_REPO_PREFIX-queue" \
	--set image.tag="$IMAGE_TAG" \
	--set image.pullPolicy=IfNotPresent \
	--set queue.replication.factor="$QUEUE_REPLICAS" \
	--set queue.replication.requiredFollowerAcks=1
wait_for_workload statefulset queue-queue queue queue

print_step "Installing streamer..."
install_chart streamer "$ROOT_DIR/deploy/helm/streamer" \
	-f "$ROOT_DIR/deploy/helm/streamer/values.pvc-example.yaml" \
	--set replicaCount="$STREAMER_REPLICAS" \
	--set image.repository="$IMAGE_REPO_PREFIX-streamer" \
	--set image.tag="$IMAGE_TAG" \
	--set image.pullPolicy=IfNotPresent \
	--set csv.fileName="$CSV_FILE_NAME"
wait_for_workload statefulset streamer-streamer streamer streamer

print_step "Installing processor..."
install_chart processor "$ROOT_DIR/deploy/helm/processor" \
	--set image.repository="$IMAGE_REPO_PREFIX-processor" \
	--set image.tag="$IMAGE_TAG" \
	--set image.pullPolicy=IfNotPresent
wait_for_workload deployment processor-processor processor processor

print_step "Installing API..."
install_chart api "$ROOT_DIR/deploy/helm/api" \
	--set image.repository="$IMAGE_REPO_PREFIX-api" \
	--set image.tag="$IMAGE_TAG" \
	--set image.pullPolicy=IfNotPresent
wait_for_workload deployment api-api api api

print_step "Verifying end-to-end flow..."
wait_for_json_query "http://api-api:8080/health" '.status == "ok"' "API health"
gpu_list="$(wait_for_json_query "http://api-api:8080/api/v1/gpus" '.items | length > 0' "non-empty GPU list")"
gpu_id="$(jq -r '.items[0].id' <<<"$gpu_list")"
telemetry_response="$(wait_for_json_query "http://api-api:8080/api/v1/gpus/${gpu_id}/telemetry?window=15m&limit=5" '.items | length > 0' "GPU telemetry response")"
sample_count="$(postgres_exec 'SELECT count(*) FROM telemetry_samples;')"
published_total="$(wait_for_queue_activity published 'queue publishes')"
delivered_total="$(wait_for_queue_activity delivered 'queue deliveries')"
acked_total="$(wait_for_queue_activity acked 'queue acknowledgements')"

if ((sample_count <= 0)); then
	echo "expected telemetry samples in postgres, found: $sample_count" >&2
	exit 1
fi

jq -e --arg gpu_id "$gpu_id" '.gpu_id == $gpu_id and (.items | length > 0)' >/dev/null <<<"$telemetry_response"

echo
echo "System test passed."
echo "  GPU ID:            $gpu_id"
echo "  API sample count:  $(jq '.items | length' <<<"$telemetry_response")"
echo "  Postgres samples:  $sample_count"
echo "  Queue published:   $published_total"
echo "  Queue delivered:   $delivered_total"
echo "  Queue acked:       $acked_total"
