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

package controllers

import (
	"context"
	"encoding/json"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	batchjobv1alpha1 "github.com/ls-1801/batchjob-operator/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

func Test(t *testing.T) {

	var descriptionList [2]PodDescription
	descriptionList[0] = PodDescription{
		PodName:  "1",
		NodeName: "1",
	}

	descriptionList[1] = PodDescription{
		PodName:  "2",
		NodeName: "2",
	}

	marshal, err := json.Marshal(descriptionList)

	if err != nil {
		t.Error(err)
	}

	println(string(marshal))
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var (
	k8sClient client.Client // You'll be using this client in your tests.
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
	WS        *WebServer = nil
)

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases"), filepath.Join("..", "config", "spark")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = batchjobv1alpha1.AddToScheme(scheme.Scheme)
	err = batchjobv1alpha1.AddToSchemeSpark(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	var reconciler = &SimpleReconciler{
		Client:    k8sManager.GetClient(),
		Queue:     []batchjobv1alpha1.Simple{},
		Scheme:    k8sManager.GetScheme(),
		Waiting:   make(chan batchjobv1alpha1.Simple),
		Scheduled: make(chan batchjobv1alpha1.Simple),
		WebServer: WebServer{Client: nil},
	}

	reconciler.WebServer.Client = reconciler
	WS = &reconciler.WebServer

	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

}, 60)

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("CronJob controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		BatchJob          = "test-cronjob"
		BatchJobNamespace = "default"
		timeout           = time.Second * 10
		interval          = time.Millisecond * 250
	)
	var ImageName = "gcr.io/spark-operator/spark:v3.1.1"
	var ImagePullPolicy = "Always"
	var MainClass = "org.apache.spark.examples.SparkPi"
	var MainClassApplicationFile = "local:///opt/spark/examples/jars/spark-examples_2.12-3.1.1.jar"
	var HostPathType = v1.HostPathType("Directory")
	var Volume = v1.Volume{
		Name: "test-volume",
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: "/path",
				Type: &HostPathType,
			},
		}}
	var Volumes = []v1.Volume{Volume}
	Context("When creating the BatchJob", func() {
		It("Should put the BatchJob into the Queue", func() {
			By("By creating a new BatchJob")
			ctx := context.Background()
			batchJob := &batchjobv1alpha1.Simple{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "batchjob.gcr.io/v1alpha1",
					Kind:       "Simple",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      BatchJob,
					Namespace: BatchJobNamespace,
				},
				Spec: batchjobv1alpha1.SimpleSpec{
					Foo: "SomeThing",
					Spec: v1beta2.SparkApplicationSpec{
						Type:                "Scala",
						SparkVersion:        "3.1.1",
						Mode:                "cluster",
						Image:               &ImageName,
						ImagePullPolicy:     &ImagePullPolicy,
						MainClass:           &MainClass,
						MainApplicationFile: &MainClassApplicationFile,
						Volumes:             Volumes,
					},
				},
			}
			Expect(k8sClient.Create(ctx, batchJob)).Should(Succeed())

			By("Inspecting the Queue using the HTTP Handler")
			Eventually(func() ([]batchjobv1alpha1.Simple, error) {
				req, err := http.NewRequest("GET", "/health-check", nil)
				Expect(err).NotTo(HaveOccurred())
				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(WS.GetQueue)
				handler.ServeHTTP(rr, req)

				// Check the status code is what we expect.
				Expect(rr.Code).Should(BeEquivalentTo(http.StatusOK))

				// Check the response body is what we expect.
				Expect(rr.Body.String()).ShouldNot(BeEmpty())

				var array = []batchjobv1alpha1.Simple{}
				err = json.Unmarshal(rr.Body.Bytes(), &array)

				return array, err
			}, timeout, interval).Should(
				And(
					WithTransform(func(p []batchjobv1alpha1.Simple) int { return len(p) },
						BeIdenticalTo(1)),
					WithTransform(func(p []batchjobv1alpha1.Simple) string { return p[0].Name },
						Equal(batchJob)),
				))

			batchjobLookupKey := types.NamespacedName{Name: BatchJob, Namespace: BatchJobNamespace}
			createdBatchJob := &batchjobv1alpha1.Simple{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, batchjobLookupKey, createdBatchJob)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(createdBatchJob.Spec.Foo).Should(Equal("SomeThing"))

			By("By checking the BatchJob is in Queue")
			Eventually(func() (bool, error) {
				err := k8sClient.Get(ctx, batchjobLookupKey, createdBatchJob)
				if err != nil {
					return false, err
				}
				return createdBatchJob.Status.InQueue, nil
			}, timeout, interval).Should(Equal(true))
		})
	})
})
