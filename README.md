# Kubernetes Resource Policy Operator

A Kubernetes operator that enforces resource policies on deployments using Open Policy Agent (OPA).

## Overview

This operator validates Kubernetes deployments against custom resource policies that define:
- Target deployments (namespace + deployment name)
- Resource limits (RAM, CPU, disk)  
- Custom OPA Rego policies

## Key Features

- **Target-based validation**: Policies target specific deployments rather than excluding namespaces
- **Dynamic policies**: OPA Rego policies are stored in CRDs, not hardcoded
- **Deployment-level validation**: Validates at deployment creation/update time
- **Multi-resource limits**: Supports RAM, CPU, and disk limits
- **Admission webhook**: Prevents policy violations before deployment creation

## Quick Start

### 1. Create Kind cluster and deploy operator

```bash
# Create Kind cluster (if not exists)
make kind-create

# Build and deploy operator
make docker-build
make load-image  
make deploy
```

### 2. Create a resource policy

```yaml
apiVersion: policies.example.com/v1alpha1
kind: ResourcePolicy
metadata:
  name: web-app-policy
spec:
  targetObjects:
    - namespace: default
      deployment: web-app
    - namespace: production  
      deployment: api-server
  limits:
    ram: "512Mi"
    cpu: "500m"
    disk: "1Gi"
  policy: |
    package kubernetes.policy

    default allow = false

    allow {
        check_ram_limit
        check_cpu_limit  
        check_disk_limit
    }

    check_ram_limit {
        input.policy.spec.limits.ram != ""
        container := input.deployment.spec.template.spec.containers[_]
        container_ram := container.resources.limits.memory
        policy_ram := input.policy.spec.limits.ram
        container_ram_bytes := units.parse_bytes(container_ram)
        policy_ram_bytes := units.parse_bytes(policy_ram)
        container_ram_bytes <= policy_ram_bytes
    }

    check_cpu_limit {
        input.policy.spec.limits.cpu != ""
        container := input.deployment.spec.template.spec.containers[_]
        container_cpu := container.resources.limits.cpu
        policy_cpu := input.policy.spec.limits.cpu
        container_cpu_millicores := units.parse_k8s_cpu(container_cpu)
        policy_cpu_millicores := units.parse_k8s_cpu(policy_cpu)
        container_cpu_millicores <= policy_cpu_millicores
    }

    check_disk_limit {
        input.policy.spec.limits.disk != ""
        # Custom disk validation logic
        true  # Simplified for example
    }
```

Apply the policy:
```bash
kubectl apply -f examples/resource-policy.yaml
```

### 3. Test the policy

**Valid deployment (should succeed):**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web-app
  template:
    metadata:
      labels:
        app: web-app
    spec:
      containers:
      - name: web
        image: nginx:latest
        resources:
          limits:
            memory: "256Mi"  # Within policy limit (512Mi)
            cpu: "250m"      # Within policy limit (500m)
```

```bash
kubectl apply -f valid-deployment.yaml  # Should succeed
```

**Invalid deployment (should be rejected):**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web-app
  template:
    metadata:
      labels:
        app: web-app
    spec:
      containers:
      - name: web
        image: nginx:latest
        resources:
          limits:
            memory: "1Gi"    # Exceeds policy limit (512Mi)
            cpu: "1000m"     # Exceeds policy limit (500m)
```

```bash
kubectl apply -f invalid-deployment.yaml  # Should be rejected by webhook
```

Expected error:
```
Error from server: admission webhook "resource-policy.policies.example.com" denied the request: Deployment violates resource policy
```

## Testing Flow

### 1. Setup and Deployment
```bash
# Create cluster and deploy
make kind-create
make docker-build && make load-image
make deploy

# Verify operator is running
kubectl get pods -n policy-system
kubectl get validatingwebhookconfigurations
```

### 2. Policy Creation and Validation
```bash
# Apply resource policy
kubectl apply -f examples/resource-policy.yaml

# Verify policy was created
kubectl get resourcepolicies
kubectl describe resourcepolicy resource-limit-policy
```

### 3. Test Valid Deployment
```bash
# Create deployment within limits
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web-app
  template:
    metadata:
      labels:
        app: web-app
    spec:
      containers:
      - name: web
        image: nginx:latest
        resources:
          limits:
            memory: "256Mi"
            cpu: "250m"
EOF

# Should succeed
kubectl get deployment web-app
```

### 4. Test Invalid Deployment  
```bash
# Try to create deployment exceeding limits
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app-invalid
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web-app-invalid
  template:
    metadata:
      labels:
        app: web-app-invalid
    spec:
      containers:
      - name: web
        image: nginx:latest
        resources:
          limits:
            memory: "1Gi"    # Exceeds 512Mi limit
            cpu: "1000m"     # Exceeds 500m limit
EOF

# Should be rejected - webhook will deny the request
```

### 5. Test Non-targeted Deployment
```bash
# Create deployment not in targetObjects (should be allowed)
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: other-app
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: other-app
  template:
    metadata:
      labels:
        app: other-app
    spec:
      containers:
      - name: web
        image: nginx:latest
        resources:
          limits:
            memory: "2Gi"    # Would exceed policy, but not targeted
            cpu: "2000m"
EOF

# Should succeed because "other-app" is not in targetObjects
```

### 6. Monitor and Debug
```bash
# Check operator logs
make logs

# Check webhook status
kubectl get validatingwebhookconfigurations resource-policy-webhook -o yaml

# Check policy status
kubectl get resourcepolicies -o yaml
```

## Architecture

- **CRD**: `ResourcePolicy` defines target deployments, limits, and OPA policies
- **Controller**: Monitors policies and deployments for compliance
- **Webhook**: Validates deployments at creation/update time using OPA
- **OPA Integration**: Executes Rego policies dynamically loaded from CRDs

## Cleanup

```bash
make undeploy
kind delete cluster
```

## Development

```bash
# Build and test locally
make build
make test

# Generate code after API changes  
make generate
controller-gen object paths="./pkg/apis/..."
```