package app

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller/deployment"
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller/kconfig"
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller/kconfigbinding"
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/server"
	clientset "github.com/gbraxton/kconfig/pkg/client/clientset/versioned"

	// _ "github.com/gbraxton/kconfig/pkg/client/clientset/versioned/scheme"
	informers "github.com/gbraxton/kconfig/pkg/client/informers/externalversions"
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

	kconfigcontroller := kconfig.NewController(
		stdClient,
		kconfigClient,
		stdInformerFactory.Core().V1().ConfigMaps(),
		stdInformerFactory.Core().V1().Secrets(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().Kconfigs(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().KconfigBindings(),
	)

	kconfigbindingcontroller := kconfigbinding.NewController(
		stdClient,
		kconfigClient,
		stdInformerFactory.Core().V1().ConfigMaps(),
		stdInformerFactory.Core().V1().Secrets(),
		stdInformerFactory.Apps().V1().Deployments(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().KconfigBindings(),
	)

	deploymentcontroller := deployment.NewController(
		stdClient,
		kconfigClient,
		stdInformerFactory.Apps().V1().Deployments(),
		kconfigInformerFactory.Kconfigcontroller().V1alpha1().KconfigBindings(),
	)

	go stdInformerFactory.Start(stopCh)
	go kconfigInformerFactory.Start(stopCh)

	go kconfigcontroller.Run(2, stopCh)
	go kconfigbindingcontroller.Run(2, stopCh)
	go deploymentcontroller.Run(2, stopCh)

	serverCfg := server.Cfg{Port: 8080}
	server := server.NewServer(serverCfg)
	go server.Start()
	defer server.Stop()

	<-stopCh
	klog.Infof("Shutting down main process")
}
