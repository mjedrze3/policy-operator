#!/usr/bin/env bash
set -e

kubectl apply -f nontarget-allowed-default-other.yaml
kubectl apply -f nontarget-allowed-prod-other.yaml
kubectl apply -f nontarget-allowed-staging-web.yaml
kubectl apply -f targeted-invalid-default-web-cpu.yaml
kubectl apply -f targeted-invalid-default-web-mem.yaml
kubectl apply -f targeted-invalid-prod-api-cpu.yaml
kubectl apply -f targeted-missing-limits-default-web.yaml
kubectl apply -f targeted-valid-default-web.yaml
kubectl apply -f targeted-valid-prod-api.yaml
