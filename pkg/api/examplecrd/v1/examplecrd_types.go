package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ExampleCrd struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ExampleCrdSpec `json:"spec"`
}

type ExampleCrdSpec struct {
	Message string `json:"message"`
	Count int `json:"count"`
}

type ExampleCrdList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ExampleCrd `json:"items"`
}