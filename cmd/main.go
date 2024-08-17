/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/kuoss/ingress-annotator/controller"
	"github.com/kuoss/ingress-annotator/pkg/rulesstore"
	// +kubebuilder:scaffold:imports
)

var (
	configMapName = "ingress-annotator"
	scheme        = runtime.NewScheme()
	setupLog      = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), getManagerOptions())
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	if err := run(mgr); err != nil {
		setupLog.Error(err, "unable to run the manager")
		os.Exit(1)
	}
}

func getManagerOptions() ctrl.Options {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			setupLog.Info("disabling http/2")
			c.NextProtos = []string{"http/1.1"}
		})
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})
	fmt.Println(webhookServer)

	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	return ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "annotator.ingress.kubernetes.io",
	}
}

func run(mgr ctrl.Manager) error {
	if !mgr.GetCache().WaitForCacheSync(context.Background()) {
		return fmt.Errorf("failed to wait for cache sync")
	}
	ns, exists := os.LookupEnv("POD_NAMESPACE")
	if !exists || ns == "" {
		return errors.New("POD_NAMESPACE environment variable is not set or is empty")
	}

	nn := types.NamespacedName{
		Namespace: ns,
		Name:      configMapName,
	}

	cm, err := fetchConfigMapDirectly(mgr.GetAPIReader(), nn)
	if err != nil {
		return err
	}
	rulesStore, err := rulesstore.New(cm)
	if err != nil {
		return fmt.Errorf("unable to start rules store: %w", err)
	}

	if err = (&controller.ConfigMapReconciler{
		Client:     mgr.GetClient(),
		NN:         nn,
		RulesStore: rulesStore,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create ConfigMapReconciler: %w", err)
	}
	if err = (&controller.IngressReconciler{
		Client:     mgr.GetClient(),
		RulesStore: rulesStore,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create IngressReconciler: %w", err)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}

func fetchConfigMapDirectly(reader client.Reader, nn types.NamespacedName) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := reader.Get(context.Background(), nn, cm)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ConfigMap: %w", err)
	}
	return cm, nil
}
