/*
Copyright 2017 The Kubernetes Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StaticEgressIP is a specification for a StaticEgressIP resource
type StaticEgressIP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StaticEgressIPSpec   `json:"spec"`
	Status StaticEgressIPStatus `json:"status"`
}

// StaticEgressIPSpec is the spec for a StaticEgressIP resource
type StaticEgressIPSpec struct {
	Rules []Rule `json:"rules"`
}

type Rule struct {
	ServiceName string `json:"service-name"`
	EgressIP    string `json:"egressip"`
	Cidr        string `json:"cidr"`
}

// StaticEgressIPStatus is the status for a StaticEgressIP resource
type StaticEgressIPStatus struct {
	GatewayNode string `json:"gateway-node"`
	GatewayIP string `json:"gateway-ip"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StaticEgressIPList is a list of StaticEgressIP resources
type StaticEgressIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []StaticEgressIP `json:"items"`
}
