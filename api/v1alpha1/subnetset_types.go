/* Copyright © 2021 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SubnetSetStatusCondition string

const (
	SubnetSetReady    SubnetSetStatusCondition = "Ready"
	SubnetSetNotReady SubnetSetStatusCondition = "NotReady"
)

// SubnetSetSpec defines the desired state of SubnetSet
type SubnetSetSpec struct {
	// Size of subnet based upon estimated workload count.
	// Defaults to 64.
	// +kubebuilder:default:=64
	IPV4SubnetSize string `json:"ipv4SubnetSize"`
	// Access mode of subnet, accessible only from within VPC or from outside VPC.
	// Defaults to private.
	// +kubebuilder:default:=private
	AccessMode string `json:"accessMode"`
}

// SubnetSetCondition defines condition of SubnetSet.
type SubnetSetCondition struct {
	// Type defines condition type.
	Type SubnetSetStatusCondition `json:"type"`
	// Status defines status of condition type, True or False.
	Status corev1.ConditionStatus `json:"status"`
	// Reason shows a brief reason of condition.
	Reason string `json:"reason,omitempty"`
	// Message shows a human readable message about condition.
	Message string `json:"message,omitempty"`
}

// SubnetItem defines subnet items of SubnetSet.
type SubnetItem struct {
	LsID       string `json:"lsID"`
	SubnetCIDR string `json:"subnetCIDR"`
}

// SubnetSetStatus defines the observed state of SubnetSet
type SubnetSetStatus struct {
	Conditions []SubnetSetCondition `json:"conditions"`
	Subnets    []SubnetItem         `json:"subnets"`
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
