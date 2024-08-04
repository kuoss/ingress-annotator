/*
Copyright 2024.

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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/node/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/jmnote/ingress-annotator/pkg/rulesstore"
)

var _ = Describe("ConfigMap Controller", func() {
	Context("When reconciling a resource", func() {

		It("should successfully reconcile the resource", func() {

			// Create a fake client with initial state
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = v1alpha1.AddToScheme(scheme) // Add your custom resources to scheme if any

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			// Create a ConfigMap to be reconciled
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-configmap",
					Namespace: "default",
				},
			}
			err := fakeClient.Create(context.TODO(), cm)
			Expect(err).NotTo(HaveOccurred())

			// Set up the reconciler
			reconciler := &ConfigMapReconciler{
				Client: fakeClient,
				Scheme: scheme,
				RulesStore: &rulesstore.RulesStore{ // Mock
					ConfigMapNamespace: "default",
					ConfigMapName:      "example-configmap",
				},
			}

			// Create a request for reconciliation
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "example-configmap",
				},
			}

			// Invoke the Reconcile method
			result, err := reconciler.Reconcile(context.TODO(), req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Add more specific assertions depending on your controller's reconciliation logic
			// Example: Verify that the data in RulesStore has been updated
			updatedData := reconciler.RulesStore.GetData()
			Expect(updatedData).ToNot(BeEmpty())
		})
	})
})
