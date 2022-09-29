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

package deployable

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"
	hdplutils "github.com/hybridapp-io/ham-deployable-operator/pkg/utils"
	prulev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
	workapiv1 "github.com/open-cluster-management/api/work/v1"
)

func (r *ReconcileHybridDeployable) updateStatus(instance *corev1alpha1.Deployable) error {
	return r.Status().Update(context.TODO(), instance)
}

func (r *ReconcileHybridDeployable) updatePerDeployerStatus(instance *corev1alpha1.Deployable, deployer *prulev1alpha1.Deployer) {
	var err error
	instanceKey := &types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}
	err = r.Get(context.TODO(), *instanceKey, instance)
	if err != nil {
		klog.Error("Failed to retrieve instance ", instanceKey.String())
		return
	}
	labelmap := map[string]string{
		corev1alpha1.OutputOf: instance.Name,
	}
	listopt := &client.ListOptions{
		Namespace:     deployer.Namespace,
		LabelSelector: labels.Set(labelmap).AsSelector(),
	}
	dplystatus := corev1alpha1.PerDeployerStatus{}

	if hdplutils.IsInClusterDeployer(deployer) {
		epslist := &corev1.EndpointsList{}

		err = r.List(context.TODO(), epslist, listopt)
		if err != nil {
			klog.Info("Failed to list endpoints with error", err)
			return
		}

		for _, eps := range epslist.Items {
			ref := corev1.ObjectReference{}
			ref.SetGroupVersionKind(eps.GetObjectKind().GroupVersionKind())
			ref.Namespace = eps.Namespace
			ref.Name = eps.Name
			dplystatus.Outputs = append(dplystatus.Outputs, ref)
		}

		cfgmaplist := &corev1.ConfigMapList{}

		err = r.List(context.TODO(), cfgmaplist, listopt)
		if err != nil {
			klog.Info("Failed to list configmaps with error", err)
			return
		}

		for _, cfgmap := range cfgmaplist.Items {
			ref := corev1.ObjectReference{}
			ref.SetGroupVersionKind(cfgmap.GetObjectKind().GroupVersionKind())
			ref.Namespace = cfgmap.Namespace
			ref.Name = cfgmap.Name
			dplystatus.Outputs = append(dplystatus.Outputs, ref)
		}

		seclist := &corev1.SecretList{}

		err = r.List(context.TODO(), seclist, listopt)
		if err != nil {
			klog.Info("Failed to list secrets with error", err)
			return
		}

		for _, sec := range seclist.Items {
			ref := corev1.ObjectReference{}
			ref.SetGroupVersionKind(sec.GetObjectKind().GroupVersionKind())
			ref.Namespace = sec.Namespace
			ref.Name = sec.Name
			dplystatus.Outputs = append(dplystatus.Outputs, ref)
		}
	} else {
		manifestworklist := &workapiv1.ManifestWorkList{}
		// populate the status outputs
		err = r.List(context.TODO(), manifestworklist, listopt)
		if err != nil {
			klog.Info("Failed to list deployables with error ", err)
			return
		}

		for _, dpl := range manifestworklist.Items {
			ref := corev1.ObjectReference{}
			ref.SetGroupVersionKind(deployableGVK)
			ref.Namespace = dpl.Namespace
			ref.Name = dpl.Name
			dplystatus.Outputs = append(dplystatus.Outputs, ref)
		}

		hostinglabelmap := map[string]string{
			corev1alpha1.HostingHybridDeployable: instance.Name,
		}
		hostinglistopt := &client.ListOptions{
			Namespace:     deployer.Namespace,
			LabelSelector: labels.Set(hostinglabelmap).AsSelector(),
		}

		// populate the status ResourceUnitStatus
		err = r.List(context.TODO(), manifestworklist, hostinglistopt)
		if err != nil {
			klog.Info("Failed to list deployables with error ", err)
			return
		}

		for _, manifestwork := range manifestworklist.Items {
			if host, ok := manifestwork.Annotations[corev1alpha1.HostingHybridDeployable]; ok {
				if host == instance.Namespace+"/"+instance.Name {
					// TODO: Fix statusdplystatus.ResourceUnitStatus = manifestwork.Status.ResourceStatus.Manifests
					break
				}
			}
		}
	}

	if instance.Status.PerDeployerStatus == nil {
		instance.Status.PerDeployerStatus = make(map[string]corev1alpha1.PerDeployerStatus)
	}

	key := types.NamespacedName{
		Namespace: deployer.Namespace,
		Name:      deployer.Name,
	}
	// update the lastUpdateTime only if PerDeployerStatus has changed
	if instance.Status.PerDeployerStatus[key.String()].LastUpdateTime != nil &&
		!r.hasStatusChanged(key, instance, dplystatus) {
		dplystatus.LastUpdateTime = instance.Status.PerDeployerStatus[key.String()].LastUpdateTime
	} else {
		now := metav1.Now()
		dplystatus.LastUpdateTime = &now

	}
	instance.Status.PerDeployerStatus[key.String()] = dplystatus
}

func (r *ReconcileHybridDeployable) hasStatusChanged(key types.NamespacedName, instance *corev1alpha1.Deployable,
	dplystatus corev1alpha1.PerDeployerStatus) bool {
	if instance.Status.PerDeployerStatus != nil {
		if !reflect.DeepEqual(instance.Status.PerDeployerStatus[key.String()].Outputs, dplystatus.Outputs) {
			return true
		}

		if !reflect.DeepEqual(instance.Status.PerDeployerStatus[key.String()].Phase, dplystatus.Phase) {
			return true
		}

		if !reflect.DeepEqual(instance.Status.PerDeployerStatus[key.String()].Reason, dplystatus.Reason) {
			return true
		}

		if !reflect.DeepEqual(instance.Status.PerDeployerStatus[key.String()].Message, dplystatus.Message) {
			return true
		}

		if !reflect.DeepEqual(instance.Status.PerDeployerStatus[key.String()].ResourceStatus, dplystatus.ResourceStatus) {
			return true
		}

	}
	return false
}
