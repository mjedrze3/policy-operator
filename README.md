# Kubernetes Resource Policy Operator

A Kubernetes operator that enforces resource policies on Deployments using **Open Policy Agent (OPA)**. Policies are defined declaratively in a **CustomResourceDefinition (CRD)** and evaluated on the admission path via a **Validating Webhook**.

## Overview

This operator validates Kubernetes Deployments against custom resource policies that define:
- Targeted Deployments (namespace + deployment name)
- Resource limits (RAM, CPU, disk)
- Custom OPA Rego policies stored in CRDs

**API Group / Version:** `policies.example.com/v1alpha1`  
**CRD Kind:** `ResourcePolicy` (cluster-scoped, short name: `rp`)  
**Namespace (operator/webhook):** `policy-system`  
**Webhook service:** `policy-webhook-service` (port 443 → 9443)  
**Webhook endpoint:** `/validate-deployment`  
**Webhook configuration name:** `resource-policy-webhook`  
**Docker image (default):** `policy-operator:latest`

## Key Features

- **Target-based admission checks** — policies are bound to specific Deployments.
- **Dynamic policies in CRDs** — Rego code is kept in the cluster, not hardcoded.
- **Admission-time verification** — violations are blocked before object creation.
- **Multiple resource limits** — RAM, CPU and disk checks supported.
- **Controller-runtime integration** — webhook server + controller in one Manager.

---

## Quick Start

### 0) Prerequisites
- Docker, `kubectl`, `kind`, `openssl`
- Go toolchain (if you plan to build locally)

### 1) Create a Kind cluster & deploy the operator

From repo root (`kubernetes-operator/`):

```bash
# Create Kind cluster (if not exists)
make kind-create

# Build image and load it into Kind
make docker-build
make load-image

# Generate TLS certs, install CRD and deploy operator + webhook
make deploy
