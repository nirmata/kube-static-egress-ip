/*
Copyright 2017 Nirmata inc.

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
	"flag"
	"time"

	"github.com/golang/glog"
	clientset "github.com/nirmata/kube-static-egress-ip/pkg/client/clientset/versioned"
	informers "github.com/nirmata/kube-static-egress-ip/pkg/client/informers/externalversions"
	"github.com/nirmata/kube-static-egress-ip/pkg/controller"
	"github.com/nirmata/kube-static-egress-ip/pkg/signals"
	"github.com/nirmata/kube-static-egress-ip/pkg/version"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()
	defer glog.Flush()

	glog.Infof("Running Nirmata static egress ip controller version: " + version.Version)
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	egressipClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building egressip clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	egressipInformerFactory := informers.NewSharedInformerFactory(egressipClient, time.Second*30)
	endpointsInformerFactory := kubeInformerFactory.Core().V1().Endpoints()

	controller := controller.NewEgressIPController(kubeClient, egressipClient, endpointsInformerFactory,
		egressipInformerFactory.Staticegressips().V1alpha1().StaticEgressIPs())

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	egressipInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
