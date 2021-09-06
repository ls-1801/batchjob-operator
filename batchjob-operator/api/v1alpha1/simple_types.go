/*
Copyright 2021.

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

	v1beta2 "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SimpleSpec defines the desired state of Simple
type SimpleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Simple. Edit simple_types.go to remove/update
	Foo  string                       `json:"foo,omitempty"`
	Spec v1beta2.SparkApplicationSpec `json:"spec"`
}

// SimpleStatus defines the observed state of Simple
type SimpleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	InQueue bool                           `json:"inQueue"`
	Running bool                           `json:"running"`
	Status  v1beta2.SparkApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Simple is the Schema for the simples API
type Simple struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SimpleSpec   `json:"spec,omitempty"`
	Status SimpleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SimpleList contains a list of Simple
type SimpleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Simple `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Simple{}, &SimpleList{})
	SchemeBuilderSpark.Register(&v1beta2.SparkApplication{}, &v1beta2.SparkApplicationList{})
}

// PrometheusMonitoringEnabled returns if Prometheus monitoring is enabled or not.
func (s *Simple) PrometheusMonitoringEnabled() bool {
	return s.Spec.Spec.Monitoring != nil && s.Spec.Spec.Monitoring.Prometheus != nil
}

// HasPrometheusConfigFile returns if Prometheus monitoring uses a configuration file in the container.
func (s *Simple) HasPrometheusConfigFile() bool {
	return s.PrometheusMonitoringEnabled() &&
		s.Spec.Spec.Monitoring.Prometheus.ConfigFile != nil &&
		*s.Spec.Spec.Monitoring.Prometheus.ConfigFile != ""
}

// HasPrometheusConfig returns if Prometheus monitoring defines metricsProperties in the spec.
func (s *Simple) HasMetricsProperties() bool {
	return s.PrometheusMonitoringEnabled() &&
		s.Spec.Spec.Monitoring.MetricsProperties != nil &&
		*s.Spec.Spec.Monitoring.MetricsProperties != ""
}

// HasPrometheusConfigFile returns if Monitoring defines metricsPropertiesFile in the spec.
func (s *Simple) HasMetricsPropertiesFile() bool {
	return s.PrometheusMonitoringEnabled() &&
		s.Spec.Spec.Monitoring.MetricsPropertiesFile != nil &&
		*s.Spec.Spec.Monitoring.MetricsPropertiesFile != ""
}

// ExposeDriverMetrics returns if driver metrics should be exposed.
func (s *Simple) ExposeDriverMetrics() bool {
	return s.Spec.Spec.Monitoring != nil && s.Spec.Spec.Monitoring.ExposeDriverMetrics
}

// ExposeExecutorMetrics returns if executor metrics should be exposed.
func (s *Simple) ExposeExecutorMetrics() bool {
	return s.Spec.Spec.Monitoring != nil && s.Spec.Spec.Monitoring.ExposeExecutorMetrics
}
