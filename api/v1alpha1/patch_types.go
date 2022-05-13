/*
Copyright 2022.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// SynchronizationSpec defines the spec of the synchronization section of a Replika
type SynchronizationSpec struct {
	Time string `json:"time"`
}

// PatchSpec defines the desired state of Patch
type PatchSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// SynchronizationSpec defines the behavior of synchronization
	Synchronization SynchronizationSpec `json:"synchronization"`

	Sources   []corev1.ObjectReference `json:"sources"`
	Target    corev1.ObjectReference   `json:"target"`
	Template  string                   `json:"template"`
	PatchType types.PatchType          `json:"patchType"`
}

// PatchStatus defines the observed state of Patch
type PatchStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Namespaced,categories={patches}
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"ResourcePatched\")].status",description=""
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"ResourcePatched\")].reason",description=""
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// Patch is the Schema for the patches API
type Patch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PatchSpec   `json:"spec,omitempty"`
	Status PatchStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PatchList contains a list of Patch
type PatchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Patch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Patch{}, &PatchList{})
}
