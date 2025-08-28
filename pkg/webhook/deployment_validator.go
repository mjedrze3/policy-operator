package webhook

import (
	"context"
	"fmt"
	"net/http"

	policiesv1alpha1 "example.com/policy-operator/pkg/apis/policies/v1alpha1"
	"example.com/policy-operator/pkg/opa"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var setupLog = log.Log.WithName("webhook-deployment-validator")

type DeploymentValidator struct {
	Client    client.Client
	Validator *opa.Validator
	decoder   *admission.Decoder
}

func (v *DeploymentValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	setupLog.Info("Received webhook request",
		"name", req.Name,
		"namespace", req.Namespace,
		"operation", req.Operation)

	deployment := &appsv1.Deployment{}
	err := v.decoder.Decode(req, deployment)
	if err != nil {
		setupLog.Error(err, "Failed to decode deployment")
		return admission.Errored(http.StatusBadRequest, err)
	}
	setupLog.Info("Deployment decoded successfully", "deployment", deployment.Name)

	// Pobierz wszystkie polityki
	var policies policiesv1alpha1.ResourcePolicyList
	if err := v.Client.List(ctx, &policies); err != nil {
		setupLog.Error(err, "Failed to list policies")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	setupLog.Info("Found policies", "count", len(policies.Items))

	// Sprawdź każdą politykę
	for _, policy := range policies.Items {
		// Sprawdź czy deployment pasuje do targetObjects
		matched := false
		for _, target := range policy.Spec.TargetObjects {
			if target.Namespace == deployment.Namespace && target.Deployment == deployment.Name {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		// Standardowa walidacja
		if err := v.Validator.ValidateDeployment(ctx, deployment, &policy); err != nil {
			return admission.Denied(fmt.Sprintf("Deployment violates resource policy: %v", err))
		}
	}

	return admission.Allowed("Deployment complies with resource policies")
}

func (v *DeploymentValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
