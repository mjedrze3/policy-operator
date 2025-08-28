# Image URL to use all building/pushing image targets
IMG ?= policy-operator:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /bin/bash

.PHONY: all
all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: build
build: fmt vet
	@echo "=== Building operator ==="
	go build -o bin/manager cmd/operator/main.go

.PHONY: run
run: fmt vet
	@echo "=== Running operator locally ==="
	go run ./cmd/operator/main.go

.PHONY: docker-build
docker-build:
	@echo "=== Building Docker image ==="
	docker build -t ${IMG} .

.PHONY: kind-create
kind-create:
	@echo "=== Creating Kind cluster ==="
	kind create cluster

.PHONY: load-image
load-image: docker-build
	@echo "=== Loading image to Kind ==="
	kind load docker-image ${IMG}

.PHONY: install
install:
	@echo "=== Installing CRD ==="
	kubectl apply -f deploy/crds/policies.example.com_resourcepolicies.yaml

.PHONY: logs
logs:
	@echo "=== Operator logs ==="
	kubectl logs -n policy-system -l app=policy-operator --tail=50

.PHONY: status
status:
	@echo "=== Cluster Status ==="
	@echo "\nPods:"
	kubectl get pods -n policy-system
	@echo "\nPolicies:"
	kubectl get resourcepolicies
	@echo "\nWebhooks:"
	kubectl get validatingwebhookconfigurations

.PHONY: generate-certs
generate-certs:
	@echo "=== Generating certificates for webhook ==="
	./generate-certs.sh

.PHONY: deploy
deploy: generate-certs
	@echo "=== Deploying operator ==="
	kubectl create namespace policy-system --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f deploy/crds/
	kubectl apply -f deploy/webhook/certs/webhook-secret.yaml
	kubectl apply -f deploy/operator/rbac.yaml
	kubectl apply -f deploy/operator/deployment.yaml
	kubectl apply -f deploy/webhook/
	@echo "\n=== Waiting for operator to be ready... ==="
	kubectl wait --for=condition=Available deployment/policy-operator -n policy-system --timeout=60s

.PHONY: test-policy
test-policy:
	@echo "=== Testing policy ==="
	kubectl apply -f examples/resource-policy.yaml
	kubectl apply -f examples/test-pod-valid.yaml
	@echo "\n=== Waiting for pods to be processed... ==="
	sleep 5
	$(MAKE) status

.PHONY: undeploy
undeploy:
	@echo "=== Cleaning up resources ==="
	kubectl delete -f examples/ --ignore-not-found || true
	kubectl delete -f deploy/webhook/ --ignore-not-found || true
	kubectl delete -f deploy/operator/ --ignore-not-found || true
	kubectl delete namespace policy-system --ignore-not-found || true
	kubectl delete crd resourcepolicies.policies.example.com --ignore-not-found || true

# ==== ResourcePolicy test harness (merged) ====
# Usage:
#   make check TEST_DIR=examples/tests
#   make check-clean TEST_DIR=examples/tests

SHELL := /bin/bash

# Allow user override: make TEST_DIR=...
TEST_DIR ?= examples/

TARGETED_OK := \
  targeted-valid-default-web.yaml \
  targeted-valid-prod-api.yaml

TARGETED_FAIL := \
  targeted-invalid-default-web-cpu.yaml \
  targeted-invalid-default-web-mem.yaml \
  targeted-missing-limits-default-web.yaml

NONTARGET_OK := \
  nontarget-allowed-default-other.yaml \
  nontarget-allowed-prod-other.yaml \
  nontarget-allowed-staging-web.yaml

# Only define these targets if they don't already exist
ifeq (,$(filter check,$(MAKECMDGOALS)))
endif

.PHONY: check
check:
	@set -euo pipefail; \
	echo "=== Running ResourcePolicy tests (dir: $(TEST_DIR)) ==="; \
	echo ""; \
	echo "--- Targeted deployments expected to PASS ---"; \
	for f in $(TARGETED_OK); do \
	  echo "APPLY OK: $$f"; \
	  kubectl apply -f "$(TEST_DIR)/$$f" >/dev/null; \
	done; \
	echo ""; \
	echo "--- Targeted deployments expected to FAIL ---"; \
	for f in $(TARGETED_FAIL); do \
	  echo "APPLY FAIL (expected): $$f"; \
	  if kubectl apply -f "$(TEST_DIR)/$$f" >/tmp/k8s-test-err.$$ 2>&1; then \
	    echo "❌ Unexpected SUCCESS for $$f"; exit 1; \
	  else \
	    if grep -q "violates resource policy" /tmp/k8s-test-err.$$; then \
	      echo "✅ Denied as expected: $$f"; \
	    else \
	      echo "---- error output ----"; cat /tmp/k8s-test-err.$$; echo "----------------------"; \
	      echo "❌ Unexpected error text for $$f"; exit 1; \
	    fi; \
	  fi; \
	  rm -f /tmp/k8s-test-err.$$; \
	done; \
	echo ""; \
	echo "--- Non-targeted deployments expected to PASS ---"; \
	for f in $(NONTARGET_OK); do \
	  echo "APPLY OK: $$f"; \
	  kubectl apply -f "$(TEST_DIR)/$$f" >/dev/null; \
	done; \
	echo ""; \
	echo "=== All checks passed ==="

.PHONY: check-clean
check-clean:
	@set -e; \
	echo "=== Deleting test deployments (dir: $(TEST_DIR)) ==="; \
	for f in $(TARGETED_OK) $(TARGETED_FAIL) $(NONTARGET_OK); do \
	  kubectl delete -f "$(TEST_DIR)/$$f" --ignore-not-found >/dev/null || true; \
	done; \
	echo "=== Done ==="
