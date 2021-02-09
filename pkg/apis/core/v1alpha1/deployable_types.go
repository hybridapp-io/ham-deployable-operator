// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	dplv1 "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

var (
	// AnnotationHybridDiscovery indicates whether a resource has been created as a result of a discovery process
	AnnotationHybridDiscovery = SchemeGroupVersion.Group + "/hybrid-discovery"

	//AnnotationClusterScope indicates whether discovery should look for resources cluster wide rather then in a specific namespace
	AnnotationClusterScope = SchemeGroupVersion.Group + "/hybrid-discovery-clusterscoped"

	SourceObject = SchemeGroupVersion.Group + "/source-object"

	DeployerType = SchemeGroupVersion.Group + "/deployer-type"

	HostingDeployer = SchemeGroupVersion.Group + "/hosting-deployer"

	DeployerInCluster                = SchemeGroupVersion.Group + "/deployer-in-cluster"
	HostingHybridDeployable          = SchemeGroupVersion.Group + "/hosting-hybriddeployable"
	ControlledBy                     = SchemeGroupVersion.Group + "/controlled-by"
	OutputOf                         = SchemeGroupVersion.Group + "/output-of"
	DependencyFrom                   = SchemeGroupVersion.Group + "/dependency-from"
	HybridDeployableController       = "hybriddeployable"
	DefaultDeployerType              = "kubernetes"
	DefaultKubernetesPlacementTarget = &metav1.GroupVersionResource{
		Group:    "clusterregistry.k8s.io",
		Version:  "v1alpha1",
		Resource: "clusters",
	}
	ClusterGVK = &metav1.GroupVersionKind{
		Group:   "clusterregistry.k8s.io",
		Version: "v1alpha1",
		Kind:    "Cluster",
	}
	DeployerPlacementTarget = &metav1.GroupVersionResource{
		Group:    "core.hybridapp.io",
		Version:  "v1alpha1",
		Resource: "deployers",
	}
	DeployerGVK = &metav1.GroupVersionKind{
		Group:   "core.hybridapp.io",
		Version: "v1alpha1",
		Kind:    "Deployer",
	}
)

const (
	// HybridDiscoveryEnabled indicates whether the discovery is enabled for a resource managed by this deployable
	HybridDiscoveryEnabled = "enabled"

	// HybridDiscoveryCompleted indicates whether the discovery has been completed for resource controlled by this deployable
	HybridDiscoveryCompleted = "completed"
)

type HybridTemplate struct {
	DeployerType string                `json:"deployerType,omitempty"`
	Template     *runtime.RawExtension `json:"template"`
}

type HybridPlacement struct {
	Deployers      []corev1.ObjectReference `json:"deployers,omitempty"`
	DeployerLabels *metav1.LabelSelector    `json:"deployerLabels,omitempty"`
	PlacementRef   *corev1.ObjectReference  `json:"placementRef,omitempty"`
}

// HybridDeployableSpec defines the desired state of HybridDeployable
type DeployableSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	HybridTemplates []HybridTemplate         `json:"hybridtemplates,omitempty"`
	Placement       *HybridPlacement         `json:"placement,omitempty"`
	Dependencies    []corev1.ObjectReference `json:"dependencies,omitempty"`
}

type PerDeployerStatus struct {
	dplv1.ResourceUnitStatus `json:",inline"`
	Outputs                  []corev1.ObjectReference `json:"outputs,omitempty"`
}

// HybridDeployableStatus defines the observed state of HybridDeployable
type DeployableStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	PerDeployerStatus map[string]PerDeployerStatus `json:"perDeployerStatus,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Deployable is the Schema for the deployables API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=deployables,scope=Namespaced
// +kubebuilder:resource:path=deployables,shortName=hdpl
type Deployable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeployableSpec   `json:"spec,omitempty"`
	Status DeployableStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeployableList contains a list of Deployable
type DeployableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Deployable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Deployable{}, &DeployableList{})
}
