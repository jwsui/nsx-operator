/* Copyright © 2021 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SubnetSetSpec defines the desired state of SubnetSet
type SubnetSetSpec struct {
	// Size of subnet based upon estimated workload count.
	// Defaults to 64.
	IPV4SubnetSize string `json:"ipv4SubnetSize,omitempty"`
	// Access mode of subnet, accessible only from within VPC or from outside VPC.
	// Defaults to private.
	AccessMode string `json:"accessMode,omitempty"`
}

// SubnetSetStatus defines the observed state of SubnetSet
type SubnetSetStatus struct {
	Conditions []SubnetSetCondition `json:"conditions"`
	Subnets    []SubnetItem         `json:"subnets"`
}

type SubnetSetStatusCondition string

const (
	SubnetSetReady SubnetSetStatusCondition = "Ready"
)

type SubnetSetCondition struct {
	Type   SubnetSetStatusCondition `json:"type"`
	Status corev1.ConditionStatus   `json:"status"`
}

type SubnetItem struct {
	LsID       string `json:"lsID"`
	SubnetCIDR string `json:"subnetCIDR"`
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
