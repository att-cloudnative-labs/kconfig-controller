package app

import (
	"k8s.io/client-go/rest"
	"time"

	"github.com/spf13/cobra"

	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller/deployment"
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller/deploymentbinding"
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller/kconfig"
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller/knativeservice"
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller/knativeservicebinding"
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller/statefulset"
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller/statefulsetbinding"
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/server"
	clientset "github.com/gbraxton/kconfig/pkg/client/clientset/versioned"
	knclientset "github.com/knative/serving/pkg/client/clientset/versioned"

	// _ "github.com/gbraxton/kconfig/pkg/client/clientset/versioned/scheme"
	informers "github.com/gbraxton/kconfig/pkg/client/informers/externalversions"
	kninformers "github.com/knative/serving/pkg/client/informers/externalversions"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"k8s.io/sample-controller/pkg/signals"
)

// NewControllerManagerCommand creates a *cobra.Command object with default parameters
func NewControllerManagerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "kconfig-controller-manager",
		Long: `The Kconfig controller-manager is a daemon controller that provides the
functionality for enacting the state defined in a Kconfig and its associated
resources.`,
		Run: run,
	}
	cmd.Flags().String("master", "", "apiserver url")
	cmd.Flags().String("kubeconfig", "", "location of kubeconfig")
	return cmd
}

func run(cmd *cobra.Command, args []string) {

	masterURL, _ := cmd.Flags().GetString("master")
	kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
	knativeServicesEnabled, _ := cmd.Flags().GetBool("knative")

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	stdClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	kconfigClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kconfig clientset: %s", err.Error())
	}



	stdInformerFactory := kubeinformers.NewSharedInformerFactory(stdClient, time.Second*30)
	kconfigInformerFactory := informers.NewSharedInformerFactory(kconfigClient, time.Second*30)


	deploymentBindingController := deploymentbinding.NewController(
		stdClient,
		kconfigClient,
		stdInformerFactory.Core().V1().ConfigMaps(),
		stdInformerFactory.Core().V1().Secrets(),
		stdInformerFactory.Apps().V1().Deployments(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().DeploymentBindings(),
	)

	statefulSetBindingController := statefulsetbinding.NewController(
		stdClient,
		kconfigClient,
		stdInformerFactory.Core().V1().ConfigMaps(),
		stdInformerFactory.Core().V1().Secrets(),
		stdInformerFactory.Apps().V1().StatefulSets(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().StatefulSetBindings(),
	)



	deploymentController := deployment.NewController(
		stdClient,
		kconfigClient,
		stdInformerFactory.Apps().V1().Deployments(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().DeploymentBindings(),
	)

	statefulSetController := statefulset.NewController(
		stdClient,
		kconfigClient,
		stdInformerFactory.Apps().V1().StatefulSets(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().StatefulSetBindings(),
	)



	kconfigController := kconfig.NewController(
		stdClient,
		kconfigClient,
		stdInformerFactory.Core().V1().ConfigMaps(),
		stdInformerFactory.Core().V1().Secrets(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().Kconfigs(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().DeploymentBindings(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().StatefulSetBindings(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().KnativeServiceBindings(),
	)

	go stdInformerFactory.Start(stopCh)
	go kconfigInformerFactory.Start(stopCh)

	if knativeServicesEnabled {
		startKnativeControllers(cfg, stdClient, kconfigClient, stdInformerFactory, kconfigInformerFactory, stopCh)
	}

	go deploymentBindingController.Run(2, stopCh)
	go statefulSetBindingController.Run(2, stopCh)


	go deploymentController.Run(2, stopCh)
	go statefulSetController.Run(2, stopCh)


	go kconfigController.Run(2, stopCh)

	serverCfg := server.Cfg{Port: 8080}
	server := server.NewServer(serverCfg)
	go server.Start()
	defer server.Stop()

	<-stopCh
	klog.Infof("Shutting down main process")
}

func startKnativeControllers(cfg *rest.Config, stdClient kubernetes.Interface, kconfigClient clientset.Interface, stdInformerFactory kubeinformers.SharedInformerFactory, kconfigInformerFactory informers.SharedInformerFactory, stopCh <-chan struct{}) {
	knativeClient, err := knclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building knative clientset: %s", err.Error())
	}

	knativeInformerFactory := kninformers.NewSharedInformerFactory(knativeClient, time.Second*30)

	knativeServiceBindingController := knativeservicebinding.NewController(
		stdClient,
		kconfigClient,
		knativeClient,
		stdInformerFactory.Core().V1().ConfigMaps(),
		stdInformerFactory.Core().V1().Secrets(),
		knativeInformerFactory.Serving().V1alpha1().Services(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().KnativeServiceBindings(),
	)
	go knativeServiceBindingController.Run(2, stopCh)

	knativeServiceController := knativeservice.NewController(
		stdClient,
		knativeClient,
		kconfigClient,
		knativeInformerFactory.Serving().V1alpha1().Services(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().KnativeServiceBindings(),
	)
	go knativeServiceController.Run(2, stopCh)

	go knativeInformerFactory.Start(stopCh)
}
