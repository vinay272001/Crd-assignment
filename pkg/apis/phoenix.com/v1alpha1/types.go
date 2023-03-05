package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PipelineRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PipelineRunSpec `json:"spec,omitempty"`
	Status PipelineRunStatus `json:"status,omitempty"`
}

type PipelineRunSpec struct {
	Message string `json:"message"`
	Count int `json:"count"`
}

type PipelineRunStatus struct {
	Message string `json:"message"`
	Count int `json:"count"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PipelineRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items           []PipelineRun `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TaskRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TaskRunSpec `json:"spec,omitempty"`
	Status TaskRunStatus `json:"status,omitempty"`
}

type TaskRunSpec struct {
	Message string `json:"message"`
	Count int `json:"count"`
}

type TaskRunStatus struct {
	Message string `json:"message"`
	Count int `json:"count"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TaskRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items           []TaskRun `json:"items"`
}