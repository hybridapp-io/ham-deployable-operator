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
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"

	apis "github.com/hybridapp-io/ham-deployable-operator/pkg/apis"
	"github.com/onsi/gomega"
	managedclusterv1 "github.com/open-cluster-management/api/cluster/v1"
	manifestworkv1 "github.com/open-cluster-management/api/work/v1"
	placementv1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var cfg *rest.Config

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "deploy", "crds"),
			filepath.Join("..", "..", "..", "hack", "test"),
		},
	}

	err := apis.AddToScheme(scheme.Scheme)
	if err != nil {
		log.Fatal(err)
	}
	err = managedclusterv1.AddToScheme(scheme.Scheme)
	if err != nil {
		klog.Fatal(err)
	}
	err = placementv1.AddToScheme(scheme.Scheme)
	if err != nil {
		klog.Fatal(err)
	}
	err = manifestworkv1.AddToScheme(scheme.Scheme)
	if err != nil {
		klog.Fatal(err)
	}
	if cfg, err = t.Start(); err != nil {
		log.Fatal(err)
	}

	code := m.Run()

	t.Stop()
	os.Exit(code)
}

// SetupTestReconcile returns a reconcile.Reconcile implementation that delegates to inner and
// writes the request to requests after Reconcile is finished.
func SetupTestReconcile(inner reconcile.Reconciler) (reconcile.Reconciler, chan reconcile.Request, chan error) {
	requests := make(chan reconcile.Request)
	errors := make(chan error)
	fn := reconcile.Func(func(req reconcile.Request) (reconcile.Result, error) {
		result, err := inner.Reconcile(req)
		requests <- req
		if err != nil {
			errors <- err
		}
		return result, err
	})

	return fn, requests, errors
}

const waitgroupDelta = 1

// StartTestManager adds recFn
func StartTestManager(mgr manager.Manager, g *gomega.GomegaWithT) (chan struct{}, *sync.WaitGroup) {
	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(waitgroupDelta)

	go func() {
		defer wg.Done()
		g.Expect(mgr.Start(stop)).NotTo(gomega.HaveOccurred())
	}()

	return stop, wg
}
