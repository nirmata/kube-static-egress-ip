
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

Since only selected pod's egress traffic (as determined by the `staticegressip` custom resource) required static egress IP it is expected that routing changes needed to redirect the traffic are exclusivley applied only to those pods. To achive that `kube-static-egress-ip` uses Linix policy based routing in following manner. 

- a custom routing table named `kube-static-egress-ip` with ID 99 is created the host network 
- all the traffic from the pods running on the node that needs static egress IP are FWMARK'ed using iptable's mangle table and PREROUTING chain. Packets are marked with with `1000`
- an `ip rule add fwmark 1000 table kube-static-egress-ip` is added to force the traffic from the pods that need static egress IP to use a custom routing table over the main routing table.
- For each destination that is specified in `staticegressip` resource a routing rule is added to send traffic via `Gateway` node

### By-pass CNI masqurading

In general most CNI's masqurade the outbound the traffic from the pods to external destination to node IP. Since `kube-static-egress-ip` intended to redirect traffic to gateway node, an additional rules is added by pass CNI masqurading rule.

## Gateway

Here is what configurations done by `kube-static-egress-ip` pod running on the node to setup functionality of gateway.

### Enable forwarding

- since default policy of FORWARD chain of the filter table is to DROP the packets, we need an explicity rule to allow forwarding of pod's traffic coming from directors nodes. To achive this `kube-static-egress-ip` adds a rule in FORWRD chain of filter table to permit the pod's traffic.

### By-pass CNI masqurading

On the gateway node also CNI masqurading need to be overidden so that egress  traffic from the pods use a static egress IP as defined by the `staticegresip` custom object. Since `kube-static-egress-ip` intended to redirect traffic to gateway node, an additional rules is added by pass CNI masqurading rule.

An explicit rule is added in the POSTROUTING chaing of nat table to SNAT traffic from the pods to a staic egress IP.

There is no additional steps needed for reverse path traffic.
