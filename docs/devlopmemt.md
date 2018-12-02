
## building

- make binary
- make container
- docker push nirmata/egressip-controller:latest 

## setup

- kubectl apply -f config/crd.yaml
- kubectl apply -f config/controller.yaml
- annotate one of the nodes to act of gateway by `kubectl annotate node 192.168.56.100 "nirmata.io/staticegressips/gateway=true"`
-
## changing the API

- edit pkg/apis/egressip/v1alpha1/types.go 
- build the clientset/informers/listers ./hack/hack/update-codegen.sh
