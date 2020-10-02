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
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appv1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"

	prulev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
	dplv1 "github.com/open-cluster-management/multicloud-operators-deployable/pkg/apis/apps/v1"
	placementv1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
)

const (
	timeout  = time.Second * 10
	interval = time.Millisecond * 10
	optime   = time.Second * 1
)

var (
	clusterName      = "cluster-1"
	clusterNamespace = "cluster-1"

	clusterNS = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterNamespace,
		},
	}

	cluster = &clusterv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: clusterNamespace,
		},
	}

	deployerName      = "mydplyr"
	deployerNamespace = clusterNamespace
	deployerKey       = types.NamespacedName{
		Name:      deployerName,
		Namespace: deployerNamespace,
	}

	deployerType = "configmap"

	deployer = &prulev1alpha1.Deployer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployerKey.Name,
			Namespace: deployerKey.Namespace,
			Labels:    map[string]string{"deployer-type": deployerType},
		},
		Spec: prulev1alpha1.DeployerSpec{
			Type: deployerType,
		},
	}

	deployerSetKey = types.NamespacedName{
		Name:      clusterName,
		Namespace: clusterNamespace,
	}

	deployerInSetSpec = prulev1alpha1.DeployerSpecDescriptor{
		Key: "default/mydplyr",
		Spec: prulev1alpha1.DeployerSpec{
			Type: deployerType,
		},
	}

	deployerSet = &prulev1alpha1.DeployerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: deployerSetKey.Namespace,
		},
		Spec: prulev1alpha1.DeployerSetSpec{
			Deployers: []prulev1alpha1.DeployerSpecDescriptor{
				deployerInSetSpec,
			},
		},
	}

	placementRuleName      = "prule-1"
	placementRuleNamespace = "default"
	placementRuleKey       = types.NamespacedName{
		Name:      placementRuleName,
		Namespace: placementRuleNamespace,
	}
	placementRule = &placementv1.PlacementRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      placementRuleName,
			Namespace: placementRuleNamespace,
		},
		Spec: placementv1.PlacementRuleSpec{
			GenericPlacementFields: placementv1.GenericPlacementFields{
				ClusterSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"name": clusterName},
				},
			},
		},
	}

	payloadFoo = &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "payload",
			Namespace: "payload",
		},
		Data: map[string]string{"myconfig": "foo"},
	}

	payloadBar = &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "payload",
		},
		Data: map[string]string{"myconfig": "bar"},
	}

	hybridDeployableName      = "test-hd"
	hybridDeployableNamespace = "test-hd-ns"
	hdplNS                    = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: hybridDeployableNamespace,
		},
	}

	hybridDeployableKey = types.NamespacedName{
		Name:      hybridDeployableName,
		Namespace: hybridDeployableNamespace,
	}

	hybridDeployable = &appv1alpha1.Deployable{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hybridDeployableName,
			Namespace: hybridDeployableNamespace,
		},
		Spec: appv1alpha1.DeployableSpec{},
	}

	expectedRequest = reconcile.Request{NamespacedName: hybridDeployableKey}
)

const oneitem = 1

func TestReconcileWithDeployer(t *testing.T) {
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

	cNS := clusterNS.DeepCopy()
	g.Expect(c.Create(context.TODO(), cNS)).To(Succeed())

	hdNS := hdplNS.DeepCopy()
	g.Expect(c.Create(context.TODO(), hdNS)).To(Succeed())

	dplyr := deployer.DeepCopy()
	g.Expect(c.Create(context.TODO(), dplyr)).To(Succeed())
	defer func() {
		if err = c.Delete(context.TODO(), dplyr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

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
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	//Expect payload to be removed on hybriddeployable delete
	if err = c.Delete(context.TODO(), instance); err != nil {
		klog.Error(err)
		t.Fail()
	}

	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	time.Sleep(optime)
	g.Expect(c.Get(context.TODO(), plKey, pl)).NotTo(Succeed())
}

func TestReconcileWithDeployerLabel(t *testing.T) {
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
			DeployerLabels: &metav1.LabelSelector{
				MatchLabels: map[string]string{"deployer-type": deployerType},
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
	defer func() {
		if err = c.Delete(context.TODO(), dplyr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	//Expect  payload is created in deplyer namespace on hybriddeployable create

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
	instance.Spec.HybridTemplates[0].Template = &runtime.RawExtension{Object: payloadBar}

	g.Expect(c.Update(context.TODO(), instance)).To(Succeed())
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	plKey = types.NamespacedName{
		Name:      payloadBar.Name,
		Namespace: deployer.Namespace,
	}
	pl = &corev1.ConfigMap{}
	g.Expect(c.Get(context.TODO(), plKey, pl)).To(Succeed())

	g.Expect(pl.Data).To(Equal(payloadBar.Data))

	//Expect payload ro be removed on hybriddeployable delete
	if err = c.Delete(context.TODO(), instance); err != nil {
		klog.Error(err)
		t.Fail()
	}

	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), plKey, pl)).NotTo(Succeed())
}

func TestReconcileWithPlacementRule(t *testing.T) {
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

	dplyr := deployer.DeepCopy()
	g.Expect(c.Create(context.TODO(), dplyr)).To(Succeed())
	defer func() {
		if err = c.Delete(context.TODO(), dplyr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	dset := deployerSet.DeepCopy()
	g.Expect(c.Create(context.TODO(), dset)).To(Succeed())
	defer func() {
		if err = c.Delete(context.TODO(), dset); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()
	clstr := cluster.DeepCopy()
	g.Expect(c.Create(context.TODO(), clstr)).To(Succeed())

	defer func() {
		if err = c.Delete(context.TODO(), clstr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

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

	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	//Expect deployable is created on hybriddeployable create
	instance := hybridDeployable.DeepCopy()
	g.Expect(c.Create(context.TODO(), instance)).To(Succeed())
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	deployableList := &dplv1.DeployableList{}
	g.Expect(c.List(context.TODO(), deployableList, client.InNamespace(deployerNamespace))).To(Succeed())
	g.Expect(deployableList.Items).To(HaveLen(oneitem))
	g.Expect(deployableList.Items[0].GetGenerateName()).To(ContainSubstring("configmap-" + payloadFoo.Namespace + "-" + payloadFoo.Name + "-"))

	//status update reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	//Expect deployble is updated on hybriddeployable template update
	instance = &appv1alpha1.Deployable{}
	g.Expect(c.Get(context.TODO(), hybridDeployableKey, instance)).To(Succeed())
	instance.Spec.HybridTemplates[0].Template = &runtime.RawExtension{Object: payloadBar}
	g.Expect(c.Update(context.TODO(), instance)).To(Succeed())
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	deployableKey := types.NamespacedName{
		Name:      deployableList.Items[0].Name,
		Namespace: deployableList.Items[0].Namespace,
	}
	dpl := &dplv1.Deployable{}
	g.Expect(c.Get(context.TODO(), deployableKey, dpl)).To(Succeed())

	tpl := dpl.Spec.Template
	codecs := serializer.NewCodecFactory(mgr.GetScheme())
	tplobj, _, err := codecs.UniversalDeserializer().Decode(tpl.Raw, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	namespacedPayloadBar := payloadBar.DeepCopy()
	namespacedPayloadBar.Namespace = instance.Namespace
	g.Expect(tplobj).To(Equal(namespacedPayloadBar))

	//Expect deployable ro be removed on hybriddeployable delete
	if err = c.Delete(context.TODO(), instance); err != nil {
		klog.Error(err)
		t.Fail()
	}

	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), deployableKey, dpl)).NotTo(Succeed())
}
