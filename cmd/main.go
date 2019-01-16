package main

import (
	goflag "flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"k8s.io/klog"

	"github.com/gbraxton/kconfig/cmd/kconfig-controller-manager/app"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	klog.InitFlags(goflag.CommandLine)
	defer klog.Flush()

	command := app.NewControllerManagerCommand()
	command.Flags().AddGoFlagSet(goflag.CommandLine)

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
