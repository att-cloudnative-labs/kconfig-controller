package main

import (
	goflag "flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"k8s.io/klog"

	"github.com/att-cloudnative-labs/kconfig-controller/cmd/kconfig-controller/app"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	klog.InitFlags(goflag.CommandLine)
	defer klog.Flush()

	command := app.NewControllerManagerCommand()
	command.Flags().AddGoFlagSet(goflag.CommandLine)
	command.Flags().Bool("knative", false, "enable configuration of knative services")

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
