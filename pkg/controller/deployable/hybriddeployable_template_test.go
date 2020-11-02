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
	"time"

	appv1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"
	prulev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
	. "github.com/onsi/gomega"
	dplv1 "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
	placementv1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	payloadNamespace = "payload"

	payloadNS = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: payloadNamespace,
		},
	}

	fooDeployer = &prulev1alpha1.Deployer{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Namespace:   "default",
			Annotations: map[string]string{appv1alpha1.DeployerInCluster: "true"},
		},
		Spec: prulev1alpha1.DeployerSpec{
			Type: "foo",
		},
	}
)

func TestCreateObjectChild(t *testing.T) {
	g := NewWithT(t)

	templateInHybridDeployable := appv1alpha1.HybridTemplate{
		DeployerType: fooDeployer.Spec.Type,
		Template: &runtime.RawExtension{
			Object: payloadFoo,
		},
	}

	hybridDeployable.Spec = appv1alpha1.DeployableSpec{
		HybridTemplates: []appv1alpha1.HybridTemplate{
			templateInHybridDeployable,
		},
	}

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests, _ := SetupTestReconcile(rec)

	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	hdNS := payloadNS.DeepCopy()
	g.Expect(c.Create(context.TODO(), hdNS)).To(Succeed())

	clusterScopeDeployer := fooDeployer.DeepCopy()
	clusterScopeDeployer.Name = "cluster-" + fooDeployer.Name
	clusterScopeDeployer.Spec.Scope = apiextensions.ClusterScoped
	clusterScopeDeployer.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), clusterScopeDeployer)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), clusterScopeDeployer)

	namespaceScopeDeployer := fooDeployer.DeepCopy()
	namespaceScopeDeployer.Name = "namespace-" + fooDeployer.Name
	namespaceScopeDeployer.Spec.Scope = apiextensions.NamespaceScoped
	namespaceScopeDeployer.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), namespaceScopeDeployer)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), namespaceScopeDeployer)

	//Expect payload is created in payload namespace on hybriddeployable create
	hdpl1 := hybridDeployable.DeepCopy()
	hdpl1.SetName("cluster-" + hybridDeployable.Name)
	hdpl1.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      clusterScopeDeployer.Name,
				Namespace: clusterScopeDeployer.Namespace,
			},
		},
	}

	g.Expect(c.Create(context.TODO(), hdpl1)).To(Succeed())
	defer c.Delete(context.TODO(), hdpl1)
	g.Eventually(requests, timeout, interval).Should(Receive())

	pl1 := &corev1.ConfigMap{}
	plKey1 := types.NamespacedName{
		Name:      payloadFoo.Name,
		Namespace: payloadFoo.Namespace,
	}
	time.Sleep(optime)
	g.Expect(c.Get(context.TODO(), plKey1, pl1)).To(Succeed())

	//status update reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive())

	//Expect payload is created in payload namespace on hybriddeployable create
	hdpl2 := hybridDeployable.DeepCopy()
	hdpl1.SetName("namespace-" + hybridDeployable.Name)
	hdpl2.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      namespaceScopeDeployer.Name,
				Namespace: namespaceScopeDeployer.Namespace,
			},
		},
	}

	g.Expect(c.Create(context.TODO(), hdpl2)).To(Succeed())
	defer c.Delete(context.TODO(), hdpl2)
	g.Eventually(requests, timeout, interval).Should(Receive())

	pl2 := &corev1.ConfigMap{}
	plKey2 := types.NamespacedName{
		Name:      payloadFoo.Name,
		Namespace: namespaceScopeDeployer.Namespace,
	}
	time.Sleep(optime)
	g.Expect(c.Get(context.TODO(), plKey2, pl2)).To(Succeed())

	//status update reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive())

}

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
	recFn, requests, _ := SetupTestReconcile(rec)

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
	recFn, requests, _ := SetupTestReconcile(rec)

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

func TestUpdateTemplateNamespace(t *testing.T) {
	g := NewWithT(t)

	templateInHybridDeployable := appv1alpha1.HybridTemplate{
		DeployerType: deployerType,
		Template: &runtime.RawExtension{
			Object: payloadBar,
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
	recFn, requests, _ := SetupTestReconcile(rec)

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

	keylabel := map[string]string{
		appv1alpha1.HostingHybridDeployable: instance.Name,
	}
	dpls := &dplv1.DeployableList{}
	g.Expect(c.List(context.TODO(), dpls, &client.ListOptions{LabelSelector: labels.SelectorFromSet(keylabel)})).To(Succeed())
	g.Expect(dpls.Items).To(HaveLen(oneitem))

	uc := &unstructured.Unstructured{}
	json.Unmarshal(dpls.Items[0].Spec.Template.Raw, uc)
	g.Expect(uc.GetNamespace()).To(Equal(instance.Namespace))

	//status update reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// update the payload to a namespaced one
	g.Expect(c.Get(context.TODO(), hybridDeployableKey, instance)).To(Succeed())

	instance.Spec.HybridTemplates[0].Template = &runtime.RawExtension{Object: payloadFoo}
	g.Expect(c.Update(context.TODO(), instance)).To(Succeed())
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	g.Expect(c.List(context.TODO(), dpls, &client.ListOptions{LabelSelector: labels.SelectorFromSet(keylabel)})).To(Succeed())
	json.Unmarshal(dpls.Items[0].Spec.Template.Raw, uc)
	g.Expect(uc.GetNamespace()).To(Equal(payloadFoo.Namespace))
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
	recFn, requests, _ := SetupTestReconcile(rec)

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

func TestDeployWithUnknownObjectCRD(t *testing.T) {
	g := NewWithT(t)

	gatewaySelectorSpec := map[string]string{
		"istio": "ingressgateway",
	}
	unstructuredPayloadGateway := &unstructured.Unstructured{}
	gatewayDeployerType := deployerType
	gatewayAPIVersion := "networking.istio.io/v1alpha3"
	gatewayKind := "Gateway"
	gatewayName := "trainticket-gateway"

	unstructuredPayloadGateway.SetAPIVersion(gatewayAPIVersion)
	unstructuredPayloadGateway.SetKind(gatewayKind)
	unstructuredPayloadGateway.SetName(gatewayName)
	err := unstructured.SetNestedField(unstructuredPayloadGateway.Object, "ingressgateway", "spec", "selector", "istio")
	g.Expect(err).NotTo(HaveOccurred())

	templateInHybridDeployable := appv1alpha1.HybridTemplate{
		DeployerType: gatewayDeployerType,
		Template: &runtime.RawExtension{
			Object: unstructuredPayloadGateway,
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
	recFn, requests, errors := SetupTestReconcile(rec)

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
	instance.SetName("gateway-hdpl")
	gatewayDeployableKey := types.NamespacedName{
		Name:      "gateway-hdpl",
		Namespace: hybridDeployableNamespace,
	}
	gatewayExpectedRequest := reconcile.Request{NamespacedName: gatewayDeployableKey}

	g.Expect(c.Create(context.TODO(), instance)).To(Succeed())

	defer c.Delete(context.TODO(), instance)

	g.Eventually(requests, timeout, interval).Should(Receive(Equal(gatewayExpectedRequest)))
	time.Sleep(optime)
	select {
	case recError := <-errors:
		klog.Error(recError)
		t.Fail()
	default:
	}

	//Expect payload is updated on hybriddeployable template update
	instance = &appv1alpha1.Deployable{}
	g.Expect(c.Get(context.TODO(), gatewayDeployableKey, instance)).To(Succeed())

	uc := &unstructured.Unstructured{}
	json.Unmarshal(instance.Spec.HybridTemplates[0].Template.Raw, uc)
	payload, _, _ := unstructured.NestedMap(uc.Object, "spec", "selector")
	g.Expect(payload["istio"].(string)).To(Equal(gatewaySelectorSpec["istio"]))
}
