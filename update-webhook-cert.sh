#!/bin/bash

# Pobierz certyfikat CA z klastra
CA_BUNDLE=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}')

# Zaktualizuj plik validating-webhook.yaml
cat > deploy/webhook/validating-webhook.yaml <<EOF
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: memory-policy-webhook
webhooks:
  - name: memory-policy.policies.example.com
    clientConfig:
      service:
        name: policy-webhook-service
        namespace: policy-system
        path: "/validate-pod"
      caBundle: ${CA_BUNDLE}
    rules:
      - apiGroups: [""]
        apiVersions: ["v1"]
        operations: ["CREATE", "UPDATE"]
        resources: ["pods"]
    sideEffects: None
    admissionReviewVersions: ["v1"]
    failurePolicy: Fail
EOF