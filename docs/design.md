
# Design

_Note: This document explains the currnet design of `kube-static-egress-ip` as its implemented now. As we explore further designed is subjected to change._

## Controller

`kube-static-egress-ip` essentially is a Kubernetes CRD controller that is run as a daemonset. `kube-static-egress-ip` watches out for the add/delete/update events corresponding to the `staticegressip` custom resource object and configure each of the node in the Kubernetes cluster with either as `Director` or `Gateway` functionality.

Idea of `Gateway` node is to use it as egress point for the pod's traffic that needs a static egress IP. On the Gateway node pod's traffic is SNAT'ed to requested static egress IP. Similarly return traffic is DNAT'ed to pod's IP. Design assumes that one or more nodes can act as Gateway nodes in the cluster. In the current implementation it is expected operator to designate single node in the cluster to act as Gatewat nodes. However it's in the roadmap to extend the functionality to support multiple Gateway nodes also leader elect a set of nodes as Gateway nodes avoiding operator configuration step.

Rest of the nodes in cluster (excluding the Gateway nodes) takes persona of Director. Idea of director is to forward the traffic from the pods' that need static egress IP to the Gateway nodes.

Depending on the persona a nodes is expected to take `kube-static-egress-ip` pod running on the node configures the node to perform respective roles.

## Director

Here is what configurations done by `kube-static-egress-ip` pod running on the node to setup functionality of director.

### policy based routing

Since only selected pod's egress traffic (as determined by the `staticegressip` custom resource) required static egress IP it is expected that routing changes needed to redirect the traffic are exclusivley applied only to those pods. To achive that `kube-static-egress-ip` used Linix policy based routing. 

### By-pass CNI masqurading

## Gateway

### Enable forwarding

### SNAT

### Routing trafffic to egress IP
