# kube-static-egress-ip

Kubernetes CRD and controller to manage static egress IP addresses for workloads


***Note: Project is in alpha state. We are activley working on improving the functionality and incorporating user feedback. Please see the roadmap. You are welcome and tryout and provide feedback.***

## Overview

### Problem Statement

Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) and [Services](https://kubernetes.io/docs/concepts/services-networking/service/) provides a in-built solution for exposing services run as workloads in the cluster to external clients outside the cluster. You have fine granular control over which services are exposed, how they are exposed, who can access them etc. But what about the reverse direction? i.e) How the workloads running in the cluster can access the service outside cluster? Through egress network policies we have basic control of which pods can access what services, beyond that Kubernetes does not prescribe how the traffic is handled. Kubernetes CNI network [plug-ins](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/) provides different functionalites to handle egress traffic.

One common functionality offered across CNI is to masqurade the egress traffic from the pods running on the node to use node IP as source IP for outbound traffic. As pod IP's are not necessarily routable from outside the cluster it provides a way for pods to communicate with outside the cluster. Its not uncommon in most on-premisis and cloud deployments to restrict access (white-listing the traffic)  to the services only to a know entities for a IP's.However this poses a challenge from security perspective for the workloads runnining the cluster. As there is no predictable egress IP that is used for the outbound traffic from the pods. It is highly desirable to fine-granular control on what IP is used for outbound traffic from the workloads (set of pods) runnin on the Kubernetes cluster.

### Solution

*kube-static-egress-ip* provides functionality with which operator can define set of pods whose outbound traffic to a specified destinations is always SNAT'ed to a configured static egress IP. *kube-static-egress-ip* provides this functionality is Kubernetes native way using custom rerources.

For e.g. below is sample definition of `staticegressip` custome resource defined by *kube-static-egress-ip*. In this case all the outbound traffic from the pods belonging to service `frontend` to destination IP `4.2.2.2` will SNAT'ed to use 100.137.146.100 as source IP. So all the traffic from selected pods to 4.2.2.2 is seen as if they are all coming from 100.137.146.100

```yaml
apiVersion: staticegressips.nirmata.io/v1alpha1
kind: StaticEgressIP
metadata:
  name: eip
spec:
  rules:
  - egressip: 100.137.146.100
    service-name: frontend
    cidr: 4.2.2.2/32
```

## Getting Started

### Installation

*kube-static-egress-ip* is pretty easy to get started.

Instatll `staticegressip` custom resource definition by installing as bellow

```sh
kubectl apply -f https://raw.githubusercontent.com/nirmata/kube-static-egress-ip/master/config/crd.yaml
```

Once you have installed custom resource you need to deploy CDR controller for `staticegressip` as below.

```sh
kubectl apply -f https://raw.githubusercontent.com/nirmata/kube-static-egress-ip/master/config/controller.yaml
```


kube-static-egress-ip` run as a daemonset so you should see a pod running on each of the cluster as below.

```sh
# kubectl get nodes -o wide 
NAME             STATUS   ROLES    AGE   VERSION   INTERNAL-IP     EXTERNAL-IP   OS-IMAGE             KERNEL-VERSION      CONTAINER-RUNTIME
falnnel-node2    Ready    <none>   18h   v1.13.0   192.168.1.202   <none>        Ubuntu 16.04.5 LTS   4.4.0-116-generic   docker://17.3.2
flannel-master   Ready    master   18h   v1.13.0   192.168.1.200   <none>        Ubuntu 16.04.5 LTS   4.4.0-116-generic   docker://17.3.2
flannel-node1    Ready    <none>   18h   v1.13.0   192.168.1.201   <none>        Ubuntu 16.04.5 LTS   4.4.0-116-generic   docker://17.3.2
#
# kubectl get pods -o wide -n kube-system -l k8s-app="egressip-controller"
NAME                        READY   STATUS    RESTARTS   AGE   IP              NODE             NOMINATED NODE   READINESS GATES
egressip-controller-cpbdn   1/1     Running   0          17h   192.168.1.201   flannel-node1    <none>           <none>
egressip-controller-hf5xm   1/1     Running   0          17h   192.168.1.202   falnnel-node2    <none>           <none>
egressip-controller-xw8nh   1/1     Running   0          17h   192.168.1.200   flannel-master   <none>           <none>
```

At this point you are all set to deploy `staticegressip` objects and see things in action. Please see an [example defintion](https://github.com/nirmata/kube-static-egress-ip/blob/master/config/example1.yaml) to how to define a `staticegressip` object
