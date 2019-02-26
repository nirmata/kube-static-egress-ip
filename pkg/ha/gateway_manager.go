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

package ha

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	egressipAPI "github.com/nirmata/kube-static-egress-ip/pkg/apis/egressip/v1alpha1"
	clientset "github.com/nirmata/kube-static-egress-ip/pkg/client/clientset/versioned"
	listers "github.com/nirmata/kube-static-egress-ip/pkg/client/listers/egressip/v1alpha1"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/transport"
)

// GatewayManager decides which node to act as gateway in the cluster
// automatically detects node failures and elects new node gateway node
type GatewayManager struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// egressIPclientset is clientset for staticegressip custom resource
	egressIPclientset clientset.Interface
	// egressIPLister can list/get StaticEgressIP from the shared informer's store
	egressIPLister listers.StaticEgressIPLister
}

// NewGatewayManager returns a new GatewayManager
func NewGatewayManager(kubeclientset kubernetes.Interface, egressIPclientset clientset.Interface, egressIPLister listers.StaticEgressIPLister) *GatewayManager {
	manager := &GatewayManager{
		kubeclientset:     kubeclientset,
		egressIPclientset: egressIPclientset,
		egressIPLister:    egressIPLister,
	}
	return manager
}

func (manager *GatewayManager) Run(stopCh <-chan struct{}) error {

	// leader election uses the Kubernetes API by writing to a ConfigMap or Endpoints
	// object. Conflicting writes are detected and each client handles those actions
	// independently.
	var config *rest.Config
	var err error
	config, err = rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	// we use the ConfigMap lock type since edits to ConfigMaps are less common
	// and fewer objects in the cluster watch "all ConfigMaps" (unlike the older
	// Endpoints lock type, where quite a few system agents like the kube-proxy
	// and ingress controllers must watch endpoints).
	id := os.Getenv("POD_IP")
	lock := &resourcelock.ConfigMapLock{
		ConfigMapMeta: metav1.ObjectMeta{
			Namespace: "kube-system",
			Name:      "static-egress-ip-configmap",
		},
		Client: kubernetes.NewForConfigOrDie(config).CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	// use a Go context so we can tell the leaderelection code when we
	// want to step down
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// use a client that will stop allowing new requests once the context ends
	config.Wrap(transport.ContextCanceller(ctx, fmt.Errorf("the leader is shutting down")))
	exampleClient := kubernetes.NewForConfigOrDie(config).CoreV1()

	go func() {
		<-stopCh
		log.Printf("Received termination, signaling shutdown")
		cancel()
	}()

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock: lock,
		// IMPORTANT: you MUST ensure that any code you have that
		// is protected by the lease must terminate **before**
		// you call cancel. Otherwise, you could have a background
		// loop still running and another process could
		// get elected before your background loop finished, violating
		// the stated goal of the lease.
		ReleaseOnCancel: true,
		LeaseDuration:   60 * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				// we're notified when we start - this is where you would
				// usually put your code
				log.Printf("%s: leading the gateway manager", id)
				staticEgressIps, err := manager.egressIPLister.List(labels.Everything())
				if err != nil {
					log.Fatalf("Failed to list static egress IP custom resources from API server: %s", err.Error())
				}
				for _, staticEgressIp := range staticEgressIps {
					gatewayNode, err := manager.chooseGateway(staticEgressIp)
					if err != nil {
						log.Printf("Failed to allocate a Gateway node for static egress IP custom resource: %s due to: %s", staticEgressIp.Name, err.Error())
						continue
					}
					log.Printf("Gateway: %s is choosen for static egress ip %s\n", gatewayNode.Name, staticEgressIp.Name)
				}
			},
			OnStoppedLeading: func() {
				// we can do cleanup here, or after the RunOrDie method
				// returns
				log.Printf("%s: lost", id)
			},
		},
	})

	// because the context is closed, the client should report errors
	_, err = exampleClient.ConfigMaps("kube-system").Get("le", metav1.GetOptions{})
	if err == nil || !strings.Contains(err.Error(), "the leader is shutting down") {
		log.Fatalf("%s: expected to get an error when trying to make a client call: %v", id, err)
	}

	// we no longer hold the lease, so perform any cleanup and then
	// exit
	log.Printf("%s: done", id)
	return nil
}

func (manager *GatewayManager) chooseGateway(staticEgressIP *egressipAPI.StaticEgressIP) (*v1core.Node, error) {

	nodes, err := manager.kubeclientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("Failed to list nodes from API server: " + err.Error())
	}

	// check if staticEgressIP custom resource already has a gateway assigned
	if staticEgressIP.Status.Gateway != "" {
		for _, node := range nodes.Items {
			if staticEgressIP.Status.Gateway == node.Name {
				return &node, nil
			}
		}
	}

	return &nodes.Items[0], nil
}
