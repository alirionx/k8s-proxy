/*
Copyright 2025.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Endpoints struct {
	// +kubebuilder:validation:Required
	Ip string `json:"ip"`
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`
}
type Service struct {
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`
	// +kubebuilder:validation:Enum=ClusterIP;LoadBalancer
	// +kubebuilder:default=ClusterIP
	Type string `json:"type"`
}
type Ingress struct {
	ClassName string `json:"className,omitempty"`
	// +kubebuilder:validation:Enum=HTTP;HTTPS
	// +kubebuilder:default=HTTP
	BackendProtocol string `json:"backendProtocol"`
	// +kubebuilder:validation:Required
	Host          string `json:"host,omitempty"`
	Tls           bool   `json:"tls,omitempty"`
	ClusterIssuer string `json:"clusterIssuer,omitempty"`
}

// ProxyEntrySpec defines the desired state of ProxyEntry.
type ProxyEntrySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	Endpoints Endpoints `json:"endpoints"`
	// +kubebuilder:validation:Required
	Service Service `json:"service"`
	// +kubebuilder:validation:Required
	Ingress Ingress `json:"ingress"`
}

// ProxyEntryStatus defines the observed state of ProxyEntry.
type ProxyEntryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ProxyEntry is the Schema for the proxyentries API.
type ProxyEntry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +kubebuilder:validation:Required
	Spec   ProxyEntrySpec   `json:"spec"`
	Status ProxyEntryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProxyEntryList contains a list of ProxyEntry.
type ProxyEntryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProxyEntry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProxyEntry{}, &ProxyEntryList{})
}
