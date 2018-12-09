# kube-static-egress-ip

Kubernetes CRD and controller to manage static egress IP addresses for workloads


***Note: Project is in alpha state. We are activley working on improving the functionality and incorporating user feedback. Please see the roadmap. You are welcome and tryout and provide feedback.***

## Overview

### Problem Statement

Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) and [Services](https://kubernetes.io/docs/concepts/services-networking/service/) provides a in-built solution for exposing services run as workloads in the cluster to external clients outside the cluster. You have fine granular control over which services are exposed, how they are exposed, who can access them etc. But what about the reverse direction? i.e) How the workloads running in the cluster can access the service outside cluster? Through egress network policies we have basic control of which pods can access what services, beyond that Kubernetes does not prescribe how the traffic is handled. Kubernetes CNI network [plug-ins](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/) provides different functionalites to handle egress traffic.

One common functionality offered across CNI is to masqurade the egress traffic from the pods running on the node to use node IP as source IP for outbound traffic. As pod IP's are not necessarily routable from outside the cluster it provides a way for pods to communicate with outside the cluster. Its not uncommon in most on-premisis and cloud deployments to restrict access (white-listing the traffic)  to the services only to a know entities for a IP's.However this poses a challenge from security perspective for the workloads runnining the cluster. As there is no predictable egress IP that is used for the outbound traffic from the pods. It is highly desirable to fine-granular control on what IP is used for outbound traffic from the workloads (set of pods) runnin on the Kubernetes cluster.
