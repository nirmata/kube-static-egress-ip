/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/nirmata/kube-static-egress-ip/pkg/apis/egressip/v1alpha1"
	scheme "github.com/nirmata/kube-static-egress-ip/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// StaticEgressIPsGetter has a method to return a StaticEgressIPInterface.
// A group's client should implement this interface.
type StaticEgressIPsGetter interface {
	StaticEgressIPs(namespace string) StaticEgressIPInterface
}

// StaticEgressIPInterface has methods to work with StaticEgressIP resources.
type StaticEgressIPInterface interface {
	Create(*v1alpha1.StaticEgressIP) (*v1alpha1.StaticEgressIP, error)
	Update(*v1alpha1.StaticEgressIP) (*v1alpha1.StaticEgressIP, error)
	UpdateStatus(*v1alpha1.StaticEgressIP) (*v1alpha1.StaticEgressIP, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.StaticEgressIP, error)
	List(opts v1.ListOptions) (*v1alpha1.StaticEgressIPList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.StaticEgressIP, err error)
	StaticEgressIPExpansion
}

// staticEgressIPs implements StaticEgressIPInterface
type staticEgressIPs struct {
	client rest.Interface
	ns     string
}

// newStaticEgressIPs returns a StaticEgressIPs
func newStaticEgressIPs(c *SamplecontrollerV1alpha1Client, namespace string) *staticEgressIPs {
	return &staticEgressIPs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the staticEgressIP, and returns the corresponding staticEgressIP object, and an error if there is any.
func (c *staticEgressIPs) Get(name string, options v1.GetOptions) (result *v1alpha1.StaticEgressIP, err error) {
	result = &v1alpha1.StaticEgressIP{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("staticegressips").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of StaticEgressIPs that match those selectors.
func (c *staticEgressIPs) List(opts v1.ListOptions) (result *v1alpha1.StaticEgressIPList, err error) {
	result = &v1alpha1.StaticEgressIPList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("staticegressips").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested staticEgressIPs.
func (c *staticEgressIPs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("staticegressips").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a staticEgressIP and creates it.  Returns the server's representation of the staticEgressIP, and an error, if there is any.
func (c *staticEgressIPs) Create(staticEgressIP *v1alpha1.StaticEgressIP) (result *v1alpha1.StaticEgressIP, err error) {
	result = &v1alpha1.StaticEgressIP{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("staticegressips").
		Body(staticEgressIP).
		Do().
		Into(result)
	return
}

// Update takes the representation of a staticEgressIP and updates it. Returns the server's representation of the staticEgressIP, and an error, if there is any.
func (c *staticEgressIPs) Update(staticEgressIP *v1alpha1.StaticEgressIP) (result *v1alpha1.StaticEgressIP, err error) {
	result = &v1alpha1.StaticEgressIP{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("staticegressips").
		Name(staticEgressIP.Name).
		Body(staticEgressIP).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *staticEgressIPs) UpdateStatus(staticEgressIP *v1alpha1.StaticEgressIP) (result *v1alpha1.StaticEgressIP, err error) {
	result = &v1alpha1.StaticEgressIP{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("staticegressips").
		Name(staticEgressIP.Name).
		SubResource("status").
		Body(staticEgressIP).
		Do().
		Into(result)
	return
}

// Delete takes name of the staticEgressIP and deletes it. Returns an error if one occurs.
func (c *staticEgressIPs) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("staticegressips").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *staticEgressIPs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("staticegressips").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched staticEgressIP.
func (c *staticEgressIPs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.StaticEgressIP, err error) {
	result = &v1alpha1.StaticEgressIP{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("staticegressips").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
