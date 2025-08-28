#!/bin/bash

# Utwórz katalog na certyfikaty
mkdir -p deploy/webhook/certs
cd deploy/webhook/certs

# Generuj klucz prywatny
openssl genrsa -out ca.key 2048

# Generuj certyfikat CA
openssl req -x509 -new -nodes -key ca.key -subj "/CN=policy-webhook-ca" -days 365 -out ca.crt

# Generuj klucz prywatny dla webhooka
openssl genrsa -out tls.key 2048

# Utwórz CSR (Certificate Signing Request)
openssl req -new -key tls.key \
    -subj "/CN=policy-webhook-service.policy-system.svc" \
    -out tls.csr

# Utwórz konfigurację dla certyfikatu
cat > cert.conf <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = policy-webhook-service
DNS.2 = policy-webhook-service.policy-system
DNS.3 = policy-webhook-service.policy-system.svc
EOF

# Podpisz certyfikat
openssl x509 -req -in tls.csr \
    -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out tls.crt \
    -days 365 \
    -extensions v3_req \
    -extfile cert.conf

# Utwórz secret z certyfikatem
kubectl create secret tls policy-webhook-certs \
    --cert=tls.crt \
    --key=tls.key \
    -n policy-system \
    --dry-run=client -o yaml > webhook-secret.yaml

# Zaktualizuj CA Bundle w konfiguracji webhooka
CA_BUNDLE=$(cat ca.crt | base64 | tr -d '\n')

cat > ../validating-webhook.yaml <<EOF
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: resource-policy-webhook
webhooks:
  - name: resource-policy.policies.example.com
    clientConfig:
      service:
        name: policy-webhook-service
        namespace: policy-system
        path: "/validate-deployment"
      caBundle: ${CA_BUNDLE}
    rules:
      - apiGroups: ["apps"]
        apiVersions: ["v1"]
        operations: ["CREATE", "UPDATE"]
        resources: ["deployments"]
    sideEffects: None
    admissionReviewVersions: ["v1"]
    failurePolicy: Fail
EOF


echo "=== Certificates and webhook for deployments generated successfully ==="

cd -
