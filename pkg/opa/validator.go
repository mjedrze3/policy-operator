package opa

import (
	"context"
	"fmt"

	policiesv1alpha1 "example.com/policy-operator/pkg/apis/policies/v1alpha1"
	"github.com/open-policy-agent/opa/rego"
	appsv1 "k8s.io/api/apps/v1"
)

type Validator struct{}

func NewValidator() (*Validator, error) {
	return &Validator{}, nil
}

func (v *Validator) ValidateDeployment(ctx context.Context, deployment *appsv1.Deployment, policy *policiesv1alpha1.ResourcePolicy) error {
	// Przygotuj dane wejściowe dla OPA
	input := map[string]interface{}{
		"deployment": deployment,
		"policy":     policy,
	}

	// Przygotuj zapytanie OPA z polityką z CRD
	query, err := rego.New(
		rego.Query("data.kubernetes.policy.allow"),
		rego.Module("policy.rego", policy.Spec.Policy),
	).PrepareForEval(ctx)

	if err != nil {
		return fmt.Errorf("failed to prepare OPA query: %v", err)
	}

	// Wykonaj walidację
	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return fmt.Errorf("OPA evaluation failed: %v", err)
	}

	// Sprawdź wynik
	if len(results) == 0 || !results[0].Expressions[0].Value.(bool) {
		return fmt.Errorf("deployment %s violates resource policy %s", deployment.Name, policy.Name)
	}

	return nil
}
