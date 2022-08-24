/* Copyright © 2021 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SubnetStatusCondition string

const (
	SubnetReady    SubnetStatusCondition = "Ready"
	SubnetNotReady SubnetStatusCondition = "NotReady"
)

// SubnetSpec defines the desired state of Subnet
type SubnetSpec struct {
	// Size of subnet based upon estimated workload count.
	// Defaults to 64.
	// +kubebuilder:default:=64
	IPV4SubnetSize string `json:"ipv4SubnetSize"`
	// Access mode of subnet, accessible only from within VPC or from outside VPC.
	// Defaults to private.
	// +kubebuilder:default:=private
	AccessMode string `json:"accessMode"`
	// Subnet CIDR.
	IPAddresses string `json:"ipAddresses,omitempty"`
}

// SubnetCondition defines condition of Subnet.
type SubnetCondition struct {
	// Type defines condition type.
	Type SubnetStatusCondition `json:"type"`
	// Status defines status of condition type, True or False.
	Status corev1.ConditionStatus `json:"status"`
	// Reason shows a brief reason of condition.
	Reason string `json:"reason,omitempty"`
	// Message shows a human readable message about condition.
	Message string `json:"message,omitempty"`
}

// SubnetStatus defines the observed state of Subnet
type SubnetStatus struct {
	// Logical switch ID.
	LsID       string            `json:"lsID"`
	SubnetCIDR string            `json:"subnetCIDR"`
	conditions []SubnetCondition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Subnet is the Schema for the subnets API
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetSpec   `json:"spec,omitempty"`
	Status SubnetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SubnetList contains a list of Subnet
type SubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Subnet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Subnet{}, &SubnetList{})
}
