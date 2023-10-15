/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FwRouterSpec defines the desired state of FwRouter
type FwRouterSpec struct {
	Zones map[ZoneName]FwZoneSpec `json:"zones"`

	Extensions map[string]string `json:"extensions,omitempty"`
}

type ZoneName string

const (
	// zone追加したい際はここを編集
	Trust   ZoneName = "trust"
	Untrust ZoneName = "untrust"
)

type FwZoneSpec struct {
	Interfaces       []string   `json:"interfaces,omitempty"`
	Policy           ZonePolicy `json:"zonePolicy"`
	AllowPrefixNames []string   `json:"allowPrefixNames,omitempty"`
}

type ZonePolicy string

const (
	EstablishedOnly ZonePolicy = "established-only"
	AllPermit       ZonePolicy = "all-permit"
)

// FwRouterStatus defines the observed state of FwRouter
type FwRouterStatus struct {
	Zones map[ZoneName]FwZoneStatus `json:"zones"`

	Extensions map[string]string `json:"extensions,omitempty"`
}

type FwZoneStatus struct {
	Interfaces       []string   `json:"interfaces,omitempty"`
	Policy           ZonePolicy `json:"zonePolicy"`
	AllowPrefixNames []string   `json:"allowPrefixNames,omitempty"`
	Created          bool       `json:"created"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FwRouter is the Schema for the fwrouters API
type FwRouter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FwRouterSpec   `json:"spec,omitempty"`
	Status FwRouterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FwRouterList contains a list of FwRouter
type FwRouterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FwRouter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FwRouter{}, &FwRouterList{})
}
