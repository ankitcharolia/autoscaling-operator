/*
Copyright 2024.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AutoScalerSpec defines the desired state of AutoScaler
type AutoScalerSpec struct {
	ScaleTargetRef  ScaleTargetRef `json:"scaleTargetRef"`
	Triggers        []Trigger      `json:"triggers"`
	MinReplicaCount int32          `json:"minReplicaCount"`
}

// ScaleTargetRef defines the target to scale
type ScaleTargetRef struct {
	Name string `json:"name"`
	Type string `json:"type"` // deployment, statefulset
}

// Trigger defines the trigger for scaling
type Trigger struct {
	Type     string          `json:"type"` // cron
	Metadata TriggerMetadata `json:"metadata"`
}

// TriggerMetadata defines metadata for a scaling trigger
type TriggerMetadata struct {
	Timezone        string `json:"timezone"`
	Start           string `json:"start"`
	End             string `json:"end"`
	DesiredReplicas int32  `json:"desiredReplicas"`
}

// AutoScalerStatus defines the observed state of AutoScaler
type AutoScalerStatus struct {
	CurrentReplicas int32       `json:"currentReplicas"`
	LastScaleTime   metav1.Time `json:"lastScaleTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AutoScaler is the Schema for the autoscalers API
type AutoScaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AutoScalerSpec   `json:"spec,omitempty"`
	Status AutoScalerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AutoScalerList contains a list of AutoScaler
type AutoScalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AutoScaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AutoScaler{}, &AutoScalerList{})
}
