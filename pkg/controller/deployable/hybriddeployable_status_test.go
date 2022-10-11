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

	appv1alpha1 "github.com/hybridapp-io/ham-deployable-operator/pkg/apis/core/v1alpha1"
	prulev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
	. "github.com/onsi/gomega"
	workapiv1 "github.com/open-cluster-management/api/work/v1"
	placementv1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	hdplName      = "output"
	hdplNamespace = "output-ns"
	hdplOutputNS  = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: hdplNamespace,
		},
	}

	hdplKey = types.NamespacedName{
		Name:      hdplName,
		Namespace: hdplNamespace,
	}
	hdDeployable = &appv1alpha1.Deployable{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hdplName,
			Namespace: hdplNamespace,
		},
		Spec: appv1alpha1.DeployableSpec{},
	}

	payloadConfigMap = &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "payload",
			Namespace:   fooDeployer.Namespace,
			Labels:      map[string]string{appv1alpha1.OutputOf: hdDeployable.Name},
			Annotations: map[string]string{appv1alpha1.OutputOf: hdDeployable.Namespace + "/" + hdDeployable.Name},
		},
		Data: map[string]string{"myconfig": "foo"},
	}

	payloadEndpoint = &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Endpoints",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "payload",
			Namespace:   fooDeployer.Namespace,
			Labels:      map[string]string{appv1alpha1.OutputOf: hdDeployable.Name},
			Annotations: map[string]string{appv1alpha1.OutputOf: hdDeployable.Namespace + "/" + hdDeployable.Name},
		},
	}

	payloadSecret = &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "payload",
			Namespace:   fooDeployer.Namespace,
			Labels:      map[string]string{appv1alpha1.OutputOf: hdDeployable.Name},
			Annotations: map[string]string{appv1alpha1.OutputOf: hdDeployable.Namespace + "/" + hdDeployable.Name},
		},
	}

	templateKubernetes = appv1alpha1.HybridTemplate{
		Template: &runtime.RawExtension{
			Object: payloadConfigMap,
		},
	}
	templateConfigMap = appv1alpha1.HybridTemplate{
		DeployerType: "configmap",
		Template: &runtime.RawExtension{
			Object: payloadConfigMap,
		},
	}
	templateEndpoints = appv1alpha1.HybridTemplate{
		DeployerType: "endpoints",
		Template: &runtime.RawExtension{
			Object: payloadEndpoint,
		},
	}
	templateSecret = appv1alpha1.HybridTemplate{
		DeployerType: "secret",
		Template: &runtime.RawExtension{
			Object: payloadSecret,
		},
	}

	hpr1Name      = "output-hpr"
	hpr1Namespace = hdplNamespace
	hpr1          = &prulev1alpha1.PlacementRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hpr1Name,
			Namespace: hpr1Namespace,
		},
		Spec: prulev1alpha1.PlacementRuleSpec{
			TargetLabels: &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": clusterName},
			},
		},
	}

	mwName      = "output-manifestwork"
	mwNamespace = clusterName
	mw1         = &workapiv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mwName,
			Namespace: mwNamespace,
			Annotations: map[string]string{
				appv1alpha1.AnnotationHybridDiscovery: "true",
				appv1alpha1.HostingHybridDeployable:   hdplNamespace + "/" + hdplName,
			},
		},
		Spec: workapiv1.ManifestWorkSpec{
			Workload: workapiv1.ManifestsTemplate{
				Manifests: []workapiv1.Manifest{
					{
						runtime.RawExtension{
							Object: payloadConfigMap,
						},
					},
				},
			},
		},
	}
)

func TestDeployableStatus(t *testing.T) {
	g := NewWithT(t)
	hdDeployable.Spec = appv1alpha1.DeployableSpec{
		HybridTemplates: []appv1alpha1.HybridTemplate{
			templateConfigMap,
			templateEndpoints,
			templateSecret,
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

	hdNS := hdplOutputNS.DeepCopy()
	g.Expect(c.Create(context.TODO(), hdNS)).To(Succeed())

	// configmap deployer
	deployerConfigMap := fooDeployer.DeepCopy()
	deployerConfigMap.Name = "configmap"
	deployerConfigMap.Spec.Type = "configmap"

	deployerConfigMap.Spec.Scope = apiextensions.ClusterScoped
	deployerConfigMap.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), deployerConfigMap)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), deployerConfigMap)

	// endpoints deployer
	deployerEndpoints := fooDeployer.DeepCopy()
	deployerEndpoints.Name = "endpoints"
	deployerEndpoints.Spec.Type = "endpoints"
	deployerEndpoints.Spec.Scope = apiextensions.ClusterScoped
	deployerEndpoints.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"endpoints"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), deployerEndpoints)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), deployerEndpoints)

	// secret deployer
	deployerSecret := fooDeployer.DeepCopy()
	deployerSecret.Name = "secret"
	deployerSecret.Spec.Type = "secret"
	deployerSecret.Spec.Scope = apiextensions.ClusterScoped
	deployerSecret.Spec.Capabilities = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"*"},
		},
	}
	g.Expect(c.Create(context.TODO(), deployerSecret)).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), deployerSecret)

	//Expect payload is created in payload namespace on hdDeployable create
	hdpl := hdDeployable.DeepCopy()
	hdpl.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      deployerConfigMap.Name,
				Namespace: deployerConfigMap.Namespace,
			},
		},
	}

	g.Expect(c.Create(context.TODO(), hdpl)).To(Succeed())

	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	c.Get(context.TODO(), types.NamespacedName{Name: hdpl.Name, Namespace: hdpl.Namespace}, hdpl)
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name])).ToNot(BeNil())

	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name]).LastUpdateTime).ToNot(BeNil())
	firstUpdateTime := hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].LastUpdateTime

	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].Outputs[0].APIVersion)).To(Equal("v1"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].Outputs[0].Kind)).To(Equal("ConfigMap"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].Outputs[0].Name)).To(Equal("payload"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].Outputs[0].Namespace)).To(Equal(fooDeployer.Namespace))

	// update placement to use the endpoints deployer in addition to the configmap deployer
	hdpl.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      deployerConfigMap.Name,
				Namespace: deployerConfigMap.Namespace,
			},
			{
				Name:      deployerEndpoints.Name,
				Namespace: deployerEndpoints.Namespace,
			},
		},
	}
	g.Expect(c.Update(context.TODO(), hdpl)).To(Succeed())
	// wait for create
	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	c.Get(context.TODO(), types.NamespacedName{Name: hdpl.Name, Namespace: hdpl.Namespace}, hdpl)
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name])).ToNot(BeNil())

	// no change to the configmap hdpl status
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerConfigMap.Name].LastUpdateTime)).To(Equal(firstUpdateTime))

	hdpl.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      deployerEndpoints.Name,
				Namespace: deployerEndpoints.Namespace,
			},
		},
	}
	g.Expect(c.Update(context.TODO(), hdpl)).To(Succeed())
	// wait for create
	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())
	c.Get(context.TODO(), types.NamespacedName{Name: hdpl.Name, Namespace: hdpl.Namespace}, hdpl)
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name]).LastUpdateTime).ToNot(BeNil())

	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name].Outputs[0].APIVersion)).To(Equal("v1"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name].Outputs[0].Kind)).To(Equal("Endpoints"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name].Outputs[0].Name)).To(Equal("payload"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerEndpoints.Name].Outputs[0].Namespace)).To(Equal(fooDeployer.Namespace))

	// update placement to use the secret deployer
	hdpl.Spec.Placement = &appv1alpha1.HybridPlacement{
		Deployers: []corev1.ObjectReference{
			{
				Name:      deployerSecret.Name,
				Namespace: deployerSecret.Namespace,
			},
		},
	}
	g.Expect(c.Update(context.TODO(), hdpl)).To(Succeed())
	// wait for create
	g.Eventually(requests, timeout, interval).Should(Receive())

	// hdpl status update
	g.Eventually(requests, timeout, interval).Should(Receive())

	// secrets  also triggers output mapper reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive())

	c.Get(context.TODO(), types.NamespacedName{Name: hdpl.Name, Namespace: hdpl.Namespace}, hdpl)

	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerSecret.Name])).ToNot(BeNil())
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerSecret.Name].Outputs[0].APIVersion)).To(Equal("v1"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerSecret.Name].Outputs[0].Kind)).To(Equal("Secret"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerSecret.Name].Outputs[0].Name)).To(Equal("payload"))
	g.Expect((hdpl.Status.PerDeployerStatus["default/"+deployerSecret.Name].Outputs[0].Namespace)).To(Equal(fooDeployer.Namespace))

	c.Delete(context.TODO(), hdpl)
	g.Eventually(requests, timeout, interval).Should(Receive())

}

func TestDeployableWithChildrenStatus(t *testing.T) {
	g := NewWithT(t)

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

	// managed cluster
	clstr := cluster.DeepCopy()
	g.Expect(c.Create(context.TODO(), clstr)).To(Succeed())

	defer func() {
		if err = c.Delete(context.TODO(), clstr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	// placement rule
	hpr := hpr1.DeepCopy()
	g.Expect(c.Create(context.TODO(), hpr)).To(Succeed())

	defer func() {
		if err = c.Delete(context.TODO(), hpr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	// Update decision of pr
	hpr.Status.Decisions = []corev1.ObjectReference{
		{
			Kind:       "ManagedCluster",
			Name:       clusterName,
			APIVersion: appv1alpha1.ClusterGVK.Group + "/" + appv1alpha1.ClusterGVK.Version,
		},
	}
	g.Expect(c.Status().Update(context.TODO(), hpr)).NotTo(HaveOccurred())

	// manifestwork
	mw := mw1.DeepCopy()
	g.Expect(c.Create(context.TODO(), mw)).To(Succeed())

	defer func() {
		if err = c.Delete(context.TODO(), mw); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	// hybrid deployable
	hdpl1 := hdDeployable.DeepCopy()
	hdpl1.Spec = appv1alpha1.DeployableSpec{
		HybridTemplates: []appv1alpha1.HybridTemplate{
			templateKubernetes,
		},
		Placement: &appv1alpha1.HybridPlacement{
			PlacementRef: &corev1.ObjectReference{
				Name:      hpr1Name,
				Namespace: hpr1Namespace,
			},
		},
	}

	g.Expect(c.Create(context.TODO(), hdpl1)).To(Succeed())

	defer func() {
		if err = c.Delete(context.TODO(), hdpl1); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Eventually(requests, timeout, interval).Should(Receive())
	g.Eventually(requests, timeout, interval).Should(Receive())

	// Check that status not empty
	g.Expect(c.Get(context.TODO(), hdplKey, hdpl1)).NotTo(HaveOccurred())

	g.Expect(hdpl1.Status.PerDeployerStatus).ToNot(BeEmpty())
}

func TestDeployableStatusPropagation(t *testing.T) {
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
		DeployerType: appv1alpha1.DefaultDeployerType,
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

	clstr := cluster.DeepCopy()
	g.Expect(c.Create(context.TODO(), clstr)).To(Succeed())

	defer c.Delete(context.TODO(), clstr)

	//Pull back the placementrule and update the status subresource
	pr := &placementv1.PlacementRule{}
	g.Expect(c.Get(context.TODO(), placementRuleKey, pr)).To(Succeed())

	decisionInPlacement := placementv1.PlacementDecision{
		ClusterName: clusterName,
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

	// Fetch manifestworks
	mws := &workapiv1.ManifestWorkList{}
	g.Expect(c.List(context.TODO(), mws, &client.ListOptions{LabelSelector: labels.SelectorFromSet(keylabel)})).To(Succeed())
	g.Expect(mws.Items).To(HaveLen(oneitem))

	mw := mws.Items[0]

	// Set status to failed & set a reason
	// TODO: fix or remove
	// mw.Status.Phase = "Failed"
	// mw.Status.Reason = "TestReason"

	// Update deployable status

	g.Expect(c.Status().Update(context.TODO(), &mw)).To(Succeed())

	// expect hdpl reconciliation to happen
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// Fetch manifestwork again and ensure status has been updated
	mws = &workapiv1.ManifestWorkList{}
	g.Expect(c.List(context.TODO(), mws, &client.ListOptions{LabelSelector: labels.SelectorFromSet(keylabel)})).To(Succeed())
	g.Expect(mws.Items).To(HaveLen(oneitem))
	mw = mws.Items[0]
	// TODO: fix or remove
	// g.Expect(string(mw.Status.ResourceUnitStatus.Phase)).To(Equal("Failed"))
	// g.Expect(mw.Status.ResourceUnitStatus.Reason).To(Equal("TestReason"))

	// Fetch hybrid deployable
	c.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, instance)
	// Status should be updated on hybrid deployable
	g.Expect((instance.Status.PerDeployerStatus[deployerName+"/"+deployerNamespace].ResourceUnitStatus.Phase)).ToNot(BeNil())
	g.Expect(len(instance.Status.PerDeployerStatus)).ToNot(Equal(0))
	g.Expect(string(instance.Status.PerDeployerStatus[deployerNamespace+"/"+deployerNamespace].ResourceUnitStatus.Phase)).To(Equal("Failed"))
	g.Expect(instance.Status.PerDeployerStatus[deployerNamespace+"/"+deployerNamespace].ResourceUnitStatus.Reason).To(Equal("TestReason"))
}
