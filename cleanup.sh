#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

NAMESPACE="${NAMESPACE:-default}"

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

uninstall_release() {
	local release="$1"

	if helm status "$release" -n "$NAMESPACE" >/dev/null 2>&1; then
		helm uninstall "$release" -n "$NAMESPACE" --wait
	else
		echo "release $release not installed, skipping"
	fi
}

delete_manifest() {
	local path="$1"

	kubectl delete -n "$NAMESPACE" -f "$path" --ignore-not-found
}

require_command kubectl
require_command helm

print_step "Removing API..."
uninstall_release api

print_step "Removing processor..."
uninstall_release processor

print_step "Removing streamer..."
uninstall_release streamer

print_step "Removing queue..."
uninstall_release queue

print_step "Removing Postgres..."
uninstall_release postgres

print_step "Removing monitoring..."
delete_manifest "$ROOT_DIR/deploy/k8s/monitoring/grafana.yaml"
delete_manifest "$ROOT_DIR/deploy/k8s/monitoring/prometheus.yaml"

print_step "Removing helper resources..."
kubectl delete -n "$NAMESPACE" -f "$ROOT_DIR/deploy/k8s/debug-curl-pod.yaml" --ignore-not-found
kubectl delete -n "$NAMESPACE" pod/dcgm-metrics-loader --ignore-not-found
kubectl delete -n "$NAMESPACE" -f "$ROOT_DIR/deploy/k8s/streamer-csv-loader-pod.yaml" --ignore-not-found
kubectl delete -n "$NAMESPACE" -f "$ROOT_DIR/deploy/k8s/streamer-csv-pvc.yaml" --ignore-not-found

echo
echo "Cleanup complete."
