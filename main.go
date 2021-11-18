/*

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
	"encoding/json"
	"flag"
	"fmt"
	"github.com/att-cloudnative-labs/kconfig-controller/webhooks"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	kconfigcontrollerv1beta1 "github.com/att-cloudnative-labs/kconfig-controller/api/v1beta1"
	"github.com/att-cloudnative-labs/kconfig-controller/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = kconfigcontrollerv1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var configMapPrefix string
	var secretPrefix string
	var defaultContainerSelector string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&configMapPrefix, "configmap-prefix", "kc-", "prefix added to name of configmaps created from kconfigs")
	flag.StringVar(&secretPrefix, "secret-prefix", "kc-", "prefix added to the name of secrets created from kconfigs")
	flag.StringVar(&defaultContainerSelector, "default-container-selector", "{}", "default container selector if kconfig doesn't supply")
	flag.Parse()

	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.KconfigReconciler{
		Client:          mgr.GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("Kconfig"),
		Scheme:          mgr.GetScheme(),
		Recorder:        mgr.GetEventRecorderFor("Kconfig"),
		ConfigMapPrefix: configMapPrefix,
		SecretPrefix:    secretPrefix,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Kconfig")
		os.Exit(1)
	}
	if err = (&controllers.KconfigBindingReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("KconfigBinding"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KconfigBinding")
		os.Exit(1)
	}

	var containerSelector v1.LabelSelector
	err = json.Unmarshal([]byte(defaultContainerSelector), &containerSelector)
	if err != nil {
		setupLog.Error(err, fmt.Sprintf("error parsing default-container-selector: %s", err.Error()))
		os.Exit(1)
	}
	setupLog.Info("setting up pod config injector webhook")
	hookServer := mgr.GetWebhookServer()
	hookServer.Register("/mutate-v1-pod", &webhook.Admission{
		Handler: &webhooks.PodConfigInjector{
			Client:                   mgr.GetClient(),
			Log:                      ctrl.Log.WithName("webhooks").WithName("pod-config-injector"),
			DefaultContainerSelector: &containerSelector,
		},
	})

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
