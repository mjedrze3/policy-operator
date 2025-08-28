package policy

import (
	"context"
	"fmt"
	"time"

	policiesv1alpha1 "example.com/policy-operator/pkg/apis/policies/v1alpha1"
	"example.com/policy-operator/pkg/opa"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type ResourcePolicyReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	Validator *opa.Validator
}

func (r *ResourcePolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Pobierz politykę
	var policy policiesv1alpha1.ResourcePolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Pobierz wszystkie deploymenty w klastrze
	var deploymentList appsv1.DeploymentList
	if err := r.List(ctx, &deploymentList); err != nil {
		logger.Error(err, "nie udało się pobrać listy deploymentów")
		return ctrl.Result{}, err
	}

	// Sprawdź limity dla każdego deploymentu
	for _, deployment := range deploymentList.Items {
		// Sprawdź czy deployment pasuje do targetObjects
		matched := false
		for _, target := range policy.Spec.TargetObjects {
			if target.Namespace == deployment.Namespace && target.Deployment == deployment.Name {
				matched = true
				break
			}
		}

		if !matched {
			logger.Info("Deployment not in target list",
				"deployment", deployment.Name,
				"namespace", deployment.Namespace)
			continue
		}

		logger.Info("Checking deployment",
			"deployment", deployment.Name,
			"namespace", deployment.Namespace,
			"targetObjects", policy.Spec.TargetObjects)

		if err := r.validateDeploymentResources(ctx, &deployment, &policy); err != nil {
			logger.Error(err, "walidacja deploymentu nie powiodła się",
				"deployment", deployment.Name,
				"namespace", deployment.Namespace)
		}
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

func (r *ResourcePolicyReconciler) validateDeploymentResources(ctx context.Context, deployment *appsv1.Deployment, policy *policiesv1alpha1.ResourcePolicy) error {
	return r.Validator.ValidateDeployment(ctx, deployment, policy)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourcePolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	validator, err := opa.NewValidator()
	if err != nil {
		return fmt.Errorf("failed to create OPA validator: %v", err)
	}
	r.Validator = validator

	return ctrl.NewControllerManagedBy(mgr).
		For(&policiesv1alpha1.ResourcePolicy{}).
		Watches(
			&source.Kind{Type: &appsv1.Deployment{}},
			handler.EnqueueRequestsFromMapFunc(r.deploymentToResourcePolicy),
		).
		Complete(r)
}

func (r *ResourcePolicyReconciler) deploymentToResourcePolicy(deployment client.Object) []reconcile.Request {
	var policies policiesv1alpha1.ResourcePolicyList
	if err := r.List(context.Background(), &policies); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, policy := range policies.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: policy.Name,
			},
		})
	}
	return requests
}
