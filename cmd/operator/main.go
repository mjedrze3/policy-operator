package main

import (
	"flag"
	"os"

	policiesv1alpha1 "example.com/policy-operator/pkg/apis/policies/v1alpha1"
	"example.com/policy-operator/pkg/controller/policy"
	"example.com/policy-operator/pkg/opa"
	"example.com/policy-operator/pkg/webhook"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	admissionwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = policiesv1alpha1.AddToScheme(scheme)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var webhookPort int

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.IntVar(&webhookPort, "webhook-bind-port", 9443, "The port the webhook server binds to.")

	flag.Parse()

	ctrl.SetLogger(zap.New())
	setupLog.Info("Logger initialized")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   webhookPort,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "policy-operator-lock",
		CertDir:                "/tmp/k8s-webhook-server/serving-certs",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	setupLog.Info("Manager created successfully",
		"metricsAddr", metricsAddr,
		"probeAddr", probeAddr,
		"webhookPort", webhookPort)

	if err = (&policy.ResourcePolicyReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("resource-policy-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ResourcePolicy")
		os.Exit(1)
	}
	setupLog.Info("ResourcePolicy controller created successfully")

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Create validator
	validator, err := opa.NewValidator()
	if err != nil {
		setupLog.Error(err, "unable to create OPA validator")
		os.Exit(1)
	}
	setupLog.Info("OPA validator created successfully")

	// Check if certificates exist
	if _, err := os.Stat("/tmp/k8s-webhook-server/serving-certs/tls.crt"); err != nil {
		setupLog.Error(err, "certificate file not found")
		os.Exit(1)
	}
	if _, err := os.Stat("/tmp/k8s-webhook-server/serving-certs/tls.key"); err != nil {
		setupLog.Error(err, "key file not found")
		os.Exit(1)
	}
	setupLog.Info("Certificate files found")

	// Setup webhook
	mgr.GetWebhookServer().Port = webhookPort
	setupLog.Info("Setting up webhook server", "port", mgr.GetWebhookServer().Port, "path", "/validate-deployment")
	deploymentValidator := &webhook.DeploymentValidator{
		Client:    mgr.GetClient(),
		Validator: validator,
	}
	mgr.GetWebhookServer().Register("/validate-deployment",
		&admissionwebhook.Admission{Handler: deploymentValidator})
	setupLog.Info("Webhook registered successfully")

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager", "error", err)
		os.Exit(1)
	}
}
