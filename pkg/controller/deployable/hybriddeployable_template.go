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
	"encoding/json"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	corev1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"

	hdplutils "github.com/hybridapp-io/ham-deployable-operator/pkg/utils"
	prulev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
	dplv1 "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
)

var (
	deployableGVR = schema.GroupVersionResource{
		Group:    "apps.open-cluster-management.io",
		Version:  "v1",
		Resource: "deployables",
	}

	deployableGVK = schema.GroupVersionKind{
		Group:   "apps.open-cluster-management.io",
		Version: "v1",
		Kind:    "Deployable",
	}

	hybriddeployableGVK = schema.GroupVersionKind{
		Group:   corev1alpha1.SchemeGroupVersion.Group,
		Version: corev1alpha1.SchemeGroupVersion.Version,
		Kind:    "Deployable",
	}
)

type gvrChildrenMap map[types.NamespacedName]metav1.Object

func (r *ReconcileHybridDeployable) purgeChildren(children map[schema.GroupVersionResource]gvrChildrenMap) {
	var err error

	for gvr, gvrchildren := range children {
		klog.V(packageDetailLogLevel).Info("Deleting obsolete children in gvr: ", gvr)

		for k, obj := range gvrchildren {
			deletePolicy := metav1.DeletePropagationBackground
			klog.Info("Deleting obsolete children ", obj.GetNamespace()+"/"+obj.GetName(), "in gvr: ", gvr)
			err = r.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Delete(context.TODO(),
				obj.GetName(), metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
			rtobj := obj.(runtime.Object)
			r.eventRecorder.RecordEvent(obj.(runtime.Object), "Delete",
				"Hybrid Deployable delete resource "+
					rtobj.GetObjectKind().GroupVersionKind().String()+" "+obj.GetNamespace()+"/"+obj.GetName(), err)

			if err != nil {
				klog.Error("Failed to delete obsolete child for key: ", k, "with error: ", err)
			}
		}
	}
}

func (r *ReconcileHybridDeployable) deployResourceByDeployers(instance *corev1alpha1.Deployable, deployers []*prulev1alpha1.Deployer,
	children map[schema.GroupVersionResource]gvrChildrenMap) error {
	if instance == nil || instance.Spec.HybridTemplates == nil || deployers == nil {
		return nil
	}

	// prepare map to ease the search of template
	klog.V(packageDetailLogLevel).Info("Building template map:", instance.Spec.HybridTemplates)

	tplmap := make(map[string]*runtime.RawExtension)

	for _, tpl := range instance.Spec.HybridTemplates {
		tpltype := tpl.DeployerType
		if tpltype == "" {
			tpltype = corev1alpha1.DefaultDeployerType
		}

		tplmap[tpltype] = tpl.Template.DeepCopy()
	}

	for _, deployer := range deployers {
		template, ok := tplmap[deployer.Spec.Type]
		if !ok {
			continue
		}

		err := r.deployResourceByDeployer(instance, deployer, children, template)

		if err != nil {
			klog.Error("Failed to deploy resource by deployer, got error: ", err)
			return err
		}
	}
	return nil
}

func (r *ReconcileHybridDeployable) deployResourceByDeployer(instance *corev1alpha1.Deployable, deployer *prulev1alpha1.Deployer,
	children map[schema.GroupVersionResource]gvrChildrenMap, template *runtime.RawExtension) error {
	obj := &unstructured.Unstructured{}

	err := json.Unmarshal(template.Raw, obj)
	if err != nil {
		klog.Info("Failed to unmarshal object:\n", string(template.Raw))
		return err
	}

	gvk := obj.GetObjectKind().GroupVersionKind()

	var gvr schema.GroupVersionResource
	if !hdplutils.IsInClusterDeployer(deployer) {
		gvr = deployableGVR
	} else {
		localgvr, err := r.registerGVK(gvk)
		if err != nil {
			klog.Error("Failed to obtain gvr for gvk ", gvk, " with error: ", err)
			return err
		}
		gvr = localgvr
	}

	if err != nil {
		return err
	}

	gvrchildren := children[gvr]
	if gvrchildren == nil {
		gvrchildren = make(gvrChildrenMap)
	}

	var metaobj metav1.Object

	var key types.NamespacedName

	for k, child := range gvrchildren {

		if hdplutils.IsNamespaceScoped(deployer) && child.GetNamespace() != deployer.GetNamespace() {
			continue
		}

		anno := child.GetAnnotations()
		if anno != nil && anno[corev1alpha1.DependencyFrom] == "" {
			metaobj = child
			key = k

			break
		}
	}

	klog.V(packageDetailLogLevel).Info("Checking gvr:", gvr, " child key:", key, "object:", metaobj)

	templateobj := &unstructured.Unstructured{}

	if template.Object != nil {
		uc, err := runtime.DefaultUnstructuredConverter.ToUnstructured(template.Object)
		if err != nil {
			klog.Error("Failed to convert template object to unstructured with error: ", err)
			return err
		}

		templateobj.SetUnstructuredContent(uc)
	} else {
		err = json.Unmarshal(template.Raw, templateobj)
		if err != nil {
			klog.Info("Failed to unmashal object with error", err, ". Raw :\n", string(template.Raw))
			return nil
		}
	}

	_, err = r.deployObjectForDeployer(instance, deployer, metaobj, templateobj)
	if err != nil {
		return err
	}

	if err == nil && metaobj != nil {
		delete(gvrchildren, key)
	}

	children[gvr] = gvrchildren
	klog.V(packageDetailLogLevel).Info("gvr children:", gvrchildren)

	r.deployDependenciesByDeployer(instance, deployer, children, templateobj)
	r.updatePerDeployerStatus(instance, deployer)

	return nil
}

func (r *ReconcileHybridDeployable) deployObjectForDeployer(instance *corev1alpha1.Deployable, deployer *prulev1alpha1.Deployer,
	object metav1.Object, templateobj *unstructured.Unstructured) (metav1.Object, error) {
	var err error
	// generate deployable
	klog.V(packageDetailLogLevel).Info("Processing Deployable for deployer type ", deployer.Spec.Type, ": ", templateobj)

	if object != nil {
		klog.V(packageDetailLogLevel).Info("Updating object:", object.GetNamespace(), "/", object.GetName())

		object, err = r.updateObjectForDeployer(instance, deployer, templateobj, object)
		if err != nil {
			klog.Error("Failed to update object with error: ", err)
			return nil, err
		}

		rtobj := object.(runtime.Object)
		r.eventRecorder.RecordEvent(object.(runtime.Object), "Update",
			"Hybrid Deploybale update resource "+
				rtobj.GetObjectKind().GroupVersionKind().String()+" "+object.GetNamespace()+"/"+object.GetName(), err)
	} else {
		object, err = r.createObjectForDeployer(instance, deployer, templateobj)
		if err != nil {
			klog.Error("Failed to create new object with error: ", err)
			return nil, err
		}
		rtobj := object.(runtime.Object)
		r.eventRecorder.RecordEvent(object.(runtime.Object), "Create",
			"Hybrid Deployable create resource "+
				rtobj.GetObjectKind().GroupVersionKind().String()+" "+object.GetNamespace()+"/"+object.GetName(), err)
	}

	if err != nil {
		klog.Error("Failed to process object with error:", err)
	}

	return object, err
}

func (r *ReconcileHybridDeployable) updateObjectForDeployer(instance *corev1alpha1.Deployable, deployer *prulev1alpha1.Deployer,
	templateobj *unstructured.Unstructured, object metav1.Object) (metav1.Object, error) {
	uc, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		klog.Error("Failed to convert deployable to unstructured with error:", err)
		return nil, err
	}

	obj := &unstructured.Unstructured{}
	obj.SetUnstructuredContent(uc)
	gvk := obj.GetObjectKind().GroupVersionKind()

	gvr, err := r.registerGVK(gvk)

	if err != nil {
		klog.Error("Failed to obtain right gvr for gvk ", gvk, " with error: ", err)
		return nil, err
	}

	// handle the deployable if not controlled by discovery (remote reconciler)
	if gvr == deployableGVR {
		if !r.isDiscoveryEnabled(obj) {
			if r.isDiscoveryCompleted(obj) {
				// assume ownership of the deployable
				objAnnotations := obj.GetAnnotations()
				delete(objAnnotations, corev1alpha1.AnnotationHybridDiscovery)
				obj.SetAnnotations(objAnnotations)

				// update the instance template and remove the discovery annotation
				instanceAnnotations := instance.GetAnnotations()
				delete(instanceAnnotations, corev1alpha1.AnnotationHybridDiscovery)
				instance.SetAnnotations(instanceAnnotations)

				for index, hybridTemplate := range instance.Spec.HybridTemplates {
					if hybridTemplate.DeployerType == deployer.Spec.Type {
						if dplTemplate, _, err := unstructured.NestedMap(obj.Object, "spec", "template"); err == nil {
							uc := &unstructured.Unstructured{}
							uc.SetUnstructuredContent(dplTemplate)

							hybridTemplate.Template = &runtime.RawExtension{
								Object: uc,
							}
							instance.Spec.HybridTemplates[index] = *hybridTemplate.DeepCopy()
						}

						//update the deployable
						if obj, err = r.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Update(context.TODO(), obj, metav1.UpdateOptions{}); err != nil {
							klog.Error("Failed to update deployable ", object.GetNamespace()+"/"+object.GetName())
							return nil, err
						}

						klog.V(packageInfoLogLevel).Info("Successfully removed the discovery annotation from deployable ",
							obj.GetNamespace()+"/"+obj.GetName())
						// update the hybrid deployable
						if err = r.Update(context.TODO(), instance); err != nil {
							klog.Error("Failed to update the hybrid deployable ", instance.GetNamespace()+"/"+instance.GetName())
							return nil, err
						}

						klog.V(packageInfoLogLevel).Info("Successfully updated the spec template for hybrid deployable ",
							instance.GetNamespace()+"/"+instance.GetName())

						return obj, nil
					}
				}
			} else if currentTemplate, _, err := unstructured.NestedMap(obj.Object, "spec", "template"); err == nil {
				if templateobj.GetNamespace() == "" {
					templateobj.SetNamespace(instance.Namespace)
				}

				if reflect.DeepEqual(currentTemplate, templateobj.Object) {
					return object, nil
				}
				if err = unstructured.SetNestedMap(obj.Object, templateobj.Object, "spec", "template"); err != nil {
					klog.Error("Failed to update object ", obj.GetNamespace()+"/"+obj.GetName(), " with error: ", err)
					return object, err
				}
				klog.V(packageInfoLogLevel).Info("Successfully updated the spec template for deployable ", obj.GetNamespace()+"/"+obj.GetName())
			}
		}
	} else {
		// use the existing object as a base and copy only the labels, annotations and spec from the template object
		newobj := obj.DeepCopy()

		newLabels := obj.GetLabels()
		if newLabels == nil {
			newLabels = make(map[string]string)
		}
		for labelKey, labelValue := range templateobj.GetLabels() {
			newLabels[labelKey] = labelValue
		}
		newobj.SetLabels(newLabels)

		newAnnotations := obj.GetAnnotations()
		if newAnnotations == nil {
			newAnnotations = make(map[string]string)
		}
		for annotationKey, annotationValue := range templateobj.GetAnnotations() {
			newAnnotations[annotationKey] = annotationValue
		}
		newobj.SetAnnotations(newAnnotations)

		newspec, _, err := unstructured.NestedMap(templateobj.Object, "spec")
		if err != nil {
			klog.Error("Failed to retrieve object spec for ", obj.GetNamespace()+"/"+obj.GetName(), " with error: ", err)
			return obj, err
		}

		if err = unstructured.SetNestedMap(newobj.Object, newspec, "spec"); err != nil {
			klog.Error("Failed to update object ", obj.GetNamespace()+"/"+obj.GetName(), " with error: ", err)
			return obj, err
		}

		// some types , like ConfigMaps, use data instead of spec
		newdata, _, err := unstructured.NestedMap(templateobj.Object, "data")
		if err != nil {
			klog.Error("Failed to retrieve object spec for ", obj.GetNamespace()+"/"+obj.GetName(), " with error: ", err)
			return obj, err
		}

		if newdata != nil {
			if err = unstructured.SetNestedMap(newobj.Object, newdata, "data"); err != nil {
				klog.Error("Failed to update object ", obj.GetNamespace()+"/"+obj.GetName(), " with error: ", err)
				return obj, err
			}
		}

		obj = newobj
	}

	klog.V(packageDetailLogLevel).Info("Updating Object:", obj)
	return r.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Update(context.TODO(), obj, metav1.UpdateOptions{})
}

func (r *ReconcileHybridDeployable) createObjectForDeployer(instance *corev1alpha1.Deployable, deployer *prulev1alpha1.Deployer,
	templateobj *unstructured.Unstructured) (metav1.Object, error) {
	gvk := templateobj.GetObjectKind().GroupVersionKind()

	var gvr schema.GroupVersionResource
	if hdplutils.IsInClusterDeployer(deployer) {
		localgvr, err := r.registerGVK(gvk)
		if err != nil {
			klog.Error("Failed to obtain gvr for gvk ", gvk, " with error: ", err)
			return nil, err
		}
		gvr = localgvr
	}

	obj := templateobj.DeepCopy()
	// actual object to be created could be template object or a deployable wrapping template object
	if !hdplutils.IsInClusterDeployer(deployer) {
		dpl := &dplv1.Deployable{}

		r.prepareUnstructured(instance, templateobj)

		dpl.Spec.Template = &runtime.RawExtension{
			Object: templateobj,
		}
		uc, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dpl)

		if err != nil {
			klog.Error("Failed to convert deployable to unstructured with error:", err)
			return nil, err
		}

		gvr = deployableGVR

		obj.SetUnstructuredContent(uc)
		obj.SetGroupVersionKind(deployableGVK)
	}

	if obj.GetName() == "" {
		obj.SetGenerateName(r.genDeployableGenerateName(templateobj))
	}

	var objNamespace string
	if !hdplutils.IsNamespaceScoped(deployer) && templateobj.GetNamespace() != "" {
		objNamespace = templateobj.GetNamespace()
	} else {
		objNamespace = deployer.Namespace
	}
	obj.SetNamespace(objNamespace)
	r.prepareUnstructured(instance, obj)

	klog.V(packageDetailLogLevel).Info("Creating Object:", obj)
	return r.dynamicClient.Resource(gvr).Namespace(objNamespace).Create(context.TODO(), obj, metav1.CreateOptions{})
}

func (r *ReconcileHybridDeployable) genDeployableGenerateName(obj *unstructured.Unstructured) string {
	return hdplutils.TruncateString(strings.ToLower(obj.GetKind()+"-"+obj.GetNamespace()+"-"+obj.GetName()), corev1alpha1.GeneratedDeployableNameLength) + "-"
}

func (r *ReconcileHybridDeployable) genObjectIdentifier(metaobj metav1.Object) types.NamespacedName {
	id := types.NamespacedName{
		Namespace: metaobj.GetNamespace(),
	}

	annotations := metaobj.GetAnnotations()
	if annotations != nil && (annotations[corev1alpha1.HostingHybridDeployable] != "" || annotations[corev1alpha1.HostingDeployer] != "") {
		id.Name = annotations[corev1alpha1.HostingHybridDeployable] + "-"
	}

	if metaobj.GetGenerateName() != "" {
		id.Name += metaobj.GetGenerateName()
	} else {
		id.Name += metaobj.GetName()
	}

	return id
}

func (r *ReconcileHybridDeployable) prepareUnstructured(instance *corev1alpha1.Deployable, object *unstructured.Unstructured) {
	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[corev1alpha1.HostingHybridDeployable] = instance.Name
	labels[corev1alpha1.ControlledBy] = corev1alpha1.HybridDeployableController

	object.SetLabels(labels)

	annotations := object.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[corev1alpha1.HostingHybridDeployable] = types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}.String()

	// add the hybrid-discovery if enabled
	if hybridDiscovery, enabled := instance.GetAnnotations()[corev1alpha1.AnnotationHybridDiscovery]; enabled {
		annotations[corev1alpha1.AnnotationHybridDiscovery] = hybridDiscovery
	}

	object.SetAnnotations(annotations)

	if object.GetNamespace() == "" {
		object.SetNamespace(instance.Namespace)
	}
}

func (r *ReconcileHybridDeployable) isDiscoveryEnabled(obj *unstructured.Unstructured) bool {
	if _, enabled := obj.GetAnnotations()[corev1alpha1.AnnotationHybridDiscovery]; !enabled ||
		obj.GetAnnotations()[corev1alpha1.AnnotationHybridDiscovery] != corev1alpha1.HybridDiscoveryEnabled {
		return false
	}

	return true
}

func (r *ReconcileHybridDeployable) isDiscoveryCompleted(obj *unstructured.Unstructured) bool {
	if _, enabled := obj.GetAnnotations()[corev1alpha1.AnnotationHybridDiscovery]; !enabled ||
		obj.GetAnnotations()[corev1alpha1.AnnotationHybridDiscovery] != corev1alpha1.HybridDiscoveryCompleted {
		return false
	}

	return true
}
