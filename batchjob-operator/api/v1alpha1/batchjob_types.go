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

// BatchJobSpec defines the desired state of BatchJob
type BatchJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of BatchJob. Edit batchjob_types.go to remove/update
	Foo  string                       `json:"foo,omitempty"`
	Spec v1beta2.SparkApplicationSpec `json:"spec"`
}

// BatchJobStatus defines the observed state of BatchJob
type BatchJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State ApplicationStateType `json:"state"`
}

// ApplicationStateType represents the type of the current state of an application.
type ApplicationStateType string

const (
	NewState              ApplicationStateType = ""
	SubmittedState        ApplicationStateType = "SUBMITTED" // Waiting for Spark to be Submitted
	InQueueState          ApplicationStateType = "QUEUE"
	RunningState          ApplicationStateType = "RUNNING"
	CompletedState        ApplicationStateType = "COMPLETED"
	FailedState           ApplicationStateType = "FAILED"
	FailedSubmissionState ApplicationStateType = "SUBMISSION_FAILED"
	PendingRerunState     ApplicationStateType = "PENDING_RERUN"
	InvalidatingState     ApplicationStateType = "INVALIDATING"
	SucceedingState       ApplicationStateType = "SUCCEEDING"
	FailingState          ApplicationStateType = "FAILING"
	UnknownState          ApplicationStateType = "UNKNOWN"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BatchJob is the Schema for the batchjobs API
type BatchJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BatchJobSpec   `json:"spec,omitempty"`
	Status BatchJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BatchJobList contains a list of BatchJob
type BatchJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BatchJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BatchJob{}, &BatchJobList{})
	SchemeBuilderSpark.Register(&v1beta2.SparkApplication{}, &v1beta2.SparkApplicationList{})
}

// PrometheusMonitoringEnabled returns if Prometheus monitoring is enabled or not.
func (s *BatchJob) PrometheusMonitoringEnabled() bool {
	return s.Spec.Spec.Monitoring != nil && s.Spec.Spec.Monitoring.Prometheus != nil
}

// HasPrometheusConfigFile returns if Prometheus monitoring uses a configuration file in the container.
func (s *BatchJob) HasPrometheusConfigFile() bool {
	return s.PrometheusMonitoringEnabled() &&
		s.Spec.Spec.Monitoring.Prometheus.ConfigFile != nil &&
		*s.Spec.Spec.Monitoring.Prometheus.ConfigFile != ""
}

// HasPrometheusConfig returns if Prometheus monitoring defines metricsProperties in the spec.
func (s *BatchJob) HasMetricsProperties() bool {
	return s.PrometheusMonitoringEnabled() &&
		s.Spec.Spec.Monitoring.MetricsProperties != nil &&
		*s.Spec.Spec.Monitoring.MetricsProperties != ""
}

// HasPrometheusConfigFile returns if Monitoring defines metricsPropertiesFile in the spec.
func (s *BatchJob) HasMetricsPropertiesFile() bool {
	return s.PrometheusMonitoringEnabled() &&
		s.Spec.Spec.Monitoring.MetricsPropertiesFile != nil &&
		*s.Spec.Spec.Monitoring.MetricsPropertiesFile != ""
}

// ExposeDriverMetrics returns if driver metrics should be exposed.
func (s *BatchJob) ExposeDriverMetrics() bool {
	return s.Spec.Spec.Monitoring != nil && s.Spec.Spec.Monitoring.ExposeDriverMetrics
}

// ExposeExecutorMetrics returns if executor metrics should be exposed.
func (s *BatchJob) ExposeExecutorMetrics() bool {
	return s.Spec.Spec.Monitoring != nil && s.Spec.Spec.Monitoring.ExposeExecutorMetrics
}
