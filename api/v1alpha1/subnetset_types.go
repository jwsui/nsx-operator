/* Copyright © 2021 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SubnetSetSpec defines the desired state of SubnetSet
type SubnetSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of SubnetSet. Edit subnetset_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// SubnetSetStatus defines the observed state of SubnetSet
type SubnetSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SubnetSet is the Schema for the subnetsets API
type SubnetSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetSetSpec   `json:"spec,omitempty"`
	Status SubnetSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SubnetSetList contains a list of SubnetSet
type SubnetSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SubnetSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SubnetSet{}, &SubnetSetList{})
}
