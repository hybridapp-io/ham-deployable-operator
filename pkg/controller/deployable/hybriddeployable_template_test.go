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
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	appv1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"
	dplv1 "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
	placementv1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
)

func TestUpdateObjectChild(t *testing.T) {
	g := NewWithT(t)

	templateInHybridDeployable := appv1alpha1.HybridTemplate{
		DeployerType: deployerType,
		Template: &runtime.RawExtension{
			Object: payloadFoo,
		},
	}

	deployerInPlacement := corev1.ObjectReference{
		Name:      deployerKey.Name,
		Namespace: deployerKey.Namespace,
	}

	hybridDeployable.Spec = appv1alpha1.DeployableSpec{
		HybridTemplates: []appv1alpha1.HybridTemplate{
			templateInHybridDeployable,
		},
		Placement: &appv1alpha1.HybridPlacement{
			Deployers: []corev1.ObjectReference{
				deployerInPlacement,
			},
		},
	}

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests := SetupTestReconcile(rec)

	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	dplyr := deployer.DeepCopy()
	g.Expect(c.Create(context.TODO(), dplyr)).To(Succeed())

	defer c.Delete(context.TODO(), dplyr)

	//Expect  payload is created in deployer namespace on hybriddeployable create
	instance := hybridDeployable.DeepCopy()
	g.Expect(c.Create(context.TODO(), instance)).To(Succeed())
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	pl := &corev1.ConfigMap{}
	plKey := types.NamespacedName{
		Name:      payloadFoo.Name,
		Namespace: deployer.Namespace,
	}
	g.Expect(c.Get(context.TODO(), plKey, pl)).To(Succeed())

	//status update reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	//Expect payload is updated on hybriddeployable template update
	instance = &appv1alpha1.Deployable{}
	g.Expect(c.Get(context.TODO(), hybridDeployableKey, instance)).To(Succeed())

	// update the paylod spec and labels/annotations
	plBar := payloadBar.DeepCopy()
	plBar.Annotations = make(map[string]string)
	plBar.Labels = make(map[string]string)

	newPayloadAnnotation := "bar_annotation"
	newPayloadLabel := "bar_label"

	plBar.Annotations[newPayloadAnnotation] = newPayloadAnnotation
	plBar.Labels[newPayloadLabel] = newPayloadLabel

	instance.Spec.HybridTemplates[0].Template = &runtime.RawExtension{Object: plBar}

	g.Expect(c.Update(context.TODO(), instance)).To(Succeed())
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	plKey = types.NamespacedName{
		Name:      payloadBar.Name,
		Namespace: deployer.Namespace,
	}
	pl = &corev1.ConfigMap{}
	g.Expect(c.Get(context.TODO(), plKey, pl)).To(Succeed())

	defer c.Delete(context.TODO(), pl)
	g.Expect(pl.Data).To(Equal(payloadBar.Data))

	g.Expect(pl.Annotations[newPayloadAnnotation]).To(Equal(newPayloadAnnotation))
	g.Expect(pl.Labels[newPayloadLabel]).To(Equal(newPayloadLabel))

	//Expect payload ro be removed on hybriddeployable delete
	c.Delete(context.TODO(), instance)
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), plKey, pl)).NotTo(Succeed())
}

func TestUpdateDeployableChild(t *testing.T) {
	g := NewWithT(t)

	templateInHybridDeployable := appv1alpha1.HybridTemplate{
		DeployerType: deployerType,
		Template: &runtime.RawExtension{
			Object: payloadFoo,
		},
	}

	hybridDeployable.Spec = appv1alpha1.DeployableSpec{
		HybridTemplates: []appv1alpha1.HybridTemplate{
			templateInHybridDeployable,
		},
		Placement: &appv1alpha1.HybridPlacement{
			PlacementRef: &corev1.ObjectReference{
				Name:      placementRuleName,
				Namespace: placementRuleNamespace,
			},
		},
	}

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests := SetupTestReconcile(rec)

	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	prule := placementRule.DeepCopy()
	g.Expect(c.Create(context.TODO(), prule)).To(Succeed())

	defer c.Delete(context.TODO(), prule)

	dplyr := deployer.DeepCopy()
	g.Expect(c.Create(context.TODO(), dplyr)).To(Succeed())

	defer c.Delete(context.TODO(), dplyr)

	dset := deployerSet.DeepCopy()
	g.Expect(c.Create(context.TODO(), dset)).To(Succeed())

	defer c.Delete(context.TODO(), dset)

	clstr := cluster.DeepCopy()
	g.Expect(c.Create(context.TODO(), clstr)).To(Succeed())

	defer c.Delete(context.TODO(), clstr)

	//Pull back the placementrule and update the status subresource
	pr := &placementv1.PlacementRule{}
	g.Expect(c.Get(context.TODO(), placementRuleKey, pr)).To(Succeed())

	decisionInPlacement := placementv1.PlacementDecision{
		ClusterName:      clusterName,
		ClusterNamespace: clusterNamespace,
	}

	newpd := []placementv1.PlacementDecision{
		decisionInPlacement,
	}
	pr.Status.Decisions = newpd
	g.Expect(c.Status().Update(context.TODO(), pr.DeepCopy())).To(Succeed())

	defer c.Delete(context.TODO(), pr)

	// empty hdpl
	instance := hybridDeployable.DeepCopy()
	g.Expect(c.Create(context.TODO(), instance)).To(Succeed())

	defer c.Delete(context.TODO(), instance)

	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	keylabel := map[string]string{
		appv1alpha1.HostingHybridDeployable: instance.Name,
	}
	dpls := &dplv1.DeployableList{}
	g.Expect(c.List(context.TODO(), dpls, &client.ListOptions{LabelSelector: labels.SelectorFromSet(keylabel)})).To(Succeed())
	g.Expect(dpls.Items).To(HaveLen(oneitem))

	//status update reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	//Expect payload is updated on hybriddeployable template update
	instance = &appv1alpha1.Deployable{}
	g.Expect(c.Get(context.TODO(), hybridDeployableKey, instance)).To(Succeed())

	instance.Spec.HybridTemplates[0].Template = &runtime.RawExtension{Object: payloadBar}

	g.Expect(c.Update(context.TODO(), instance)).To(Succeed())
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	uc := &unstructured.Unstructured{}
	updatedDpls := &dplv1.DeployableList{}
	g.Expect(c.List(context.TODO(), updatedDpls, &client.ListOptions{LabelSelector: labels.SelectorFromSet(keylabel)})).To(Succeed())

	json.Unmarshal(updatedDpls.Items[0].Spec.Template.Raw, uc)
	payload, _, _ := unstructured.NestedMap(uc.Object, "data")
	g.Expect(payload["myconfig"].(string)).To(Equal(payloadBar.Data["myconfig"]))
}

func TestUpdateDiscoveryCompleted(t *testing.T) {
	g := NewWithT(t)

	payloadIncomplete := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "payload",
			Namespace: "payload",
		},
	}
	templateInHybridDeployable := appv1alpha1.HybridTemplate{
		DeployerType: deployerType,
		Template: &runtime.RawExtension{
			Object: payloadIncomplete,
		},
	}

	hybridDeployable.Spec = appv1alpha1.DeployableSpec{
		HybridTemplates: []appv1alpha1.HybridTemplate{
			templateInHybridDeployable,
		},
		Placement: &appv1alpha1.HybridPlacement{
			PlacementRef: &corev1.ObjectReference{
				Name:      placementRuleName,
				Namespace: placementRuleNamespace,
			},
		},
	}

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests := SetupTestReconcile(rec)

	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	prule := placementRule.DeepCopy()
	g.Expect(c.Create(context.TODO(), prule)).To(Succeed())

	defer c.Delete(context.TODO(), prule)

	dplyr := deployer.DeepCopy()
	g.Expect(c.Create(context.TODO(), dplyr)).To(Succeed())

	defer c.Delete(context.TODO(), dplyr)

	dset := deployerSet.DeepCopy()
	g.Expect(c.Create(context.TODO(), dset)).To(Succeed())

	defer c.Delete(context.TODO(), dset)

	clstr := cluster.DeepCopy()
	g.Expect(c.Create(context.TODO(), clstr)).To(Succeed())

	defer c.Delete(context.TODO(), clstr)

	//Pull back the placementrule and update the status subresource
	pr := &placementv1.PlacementRule{}
	g.Expect(c.Get(context.TODO(), placementRuleKey, pr)).To(Succeed())

	decisionInPlacement := placementv1.PlacementDecision{
		ClusterName:      clusterName,
		ClusterNamespace: clusterNamespace,
	}

	newpd := []placementv1.PlacementDecision{
		decisionInPlacement,
	}
	pr.Status.Decisions = newpd
	g.Expect(c.Status().Update(context.TODO(), pr.DeepCopy())).To(Succeed())

	defer c.Delete(context.TODO(), pr)

	instance := hybridDeployable.DeepCopy()
	g.Expect(c.Create(context.TODO(), instance)).To(Succeed())

	defer c.Delete(context.TODO(), instance)
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	//status update reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// connect the deployable with a payload and set its discovery annotation to enabled
	keylabel := map[string]string{
		appv1alpha1.HostingHybridDeployable: instance.Namespace + "/" + instance.Name,
	}

	dpls := &dplv1.DeployableList{}
	g.Expect(c.List(context.TODO(), dpls, &client.ListOptions{LabelSelector: labels.SelectorFromSet(keylabel)})).To(Succeed())
	g.Expect(dpls.Items).To(HaveLen(oneitem))

	dpl := dpls.Items[0]
	annotations := dpl.GetAnnotations()
	annotations[appv1alpha1.AnnotationHybridDiscovery] = appv1alpha1.HybridDiscoveryCompleted
	dpl.SetAnnotations(annotations)
	dpl.Spec = dplv1.DeployableSpec{
		Template: &runtime.RawExtension{
			Object: payloadFoo,
		},
	}
	g.Expect(c.Update(context.TODO(), &dpl)).To(Succeed())

	// expect hdpl reconciliation to happen
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	//Expect payload is updated on hybriddeployable template update
	instance = &appv1alpha1.Deployable{}
	g.Expect(c.Get(context.TODO(), hybridDeployableKey, instance)).To(Succeed())

	uc := &unstructured.Unstructured{}
	json.Unmarshal(instance.Spec.HybridTemplates[0].Template.Raw, uc)
	payload, _, _ := unstructured.NestedMap(uc.Object, "data")
	g.Expect(payload["myconfig"].(string)).To(Equal(payloadFoo.Data["myconfig"]))
}
