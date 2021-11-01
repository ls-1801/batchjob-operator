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
	"bytes"
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
	TestNode  string     = "test-node"
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
		Client:      k8sManager.GetClient(),
		Scheme:      k8sManager.GetScheme(),
		ManagedJobs: map[types.NamespacedName]*batchjobv1alpha1.Simple{},
		SparkCtrl:   nil,
		WebServer:   nil,
	}

	WS = NewWebServer(reconciler)
	reconciler.WebServer = WS

	reconciler.SparkCtrl = NewSparkController(reconciler)

	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	node := &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: TestNode,
		},
		Spec: v1.NodeSpec{
			PodCIDR: "10.0.0.0/21",
		},
	}
	Expect(k8sClient.Create(ctx, node)).Should(Succeed())

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

	Context("When creating the BatchJob", func() {

		It("Should put the BatchJob into the Queue", func() {
			By("Verifying that the Queue is Empty")
			Eventually(func() ([]JobDescription, error) {
				req, err := http.NewRequest("GET", "/queue", nil)
				Expect(err).NotTo(HaveOccurred())
				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(WS.GetQueue)
				handler.ServeHTTP(rr, req)

				// Check the status code is what we expect.
				Expect(rr.Code).Should(BeEquivalentTo(http.StatusOK))

				// Check the response body is what we expect.
				Expect(rr.Body.String()).ShouldNot(BeEmpty())

				var array []JobDescription
				err = json.Unmarshal(rr.Body.Bytes(), &array)

				return array, err
			}, timeout, interval).Should(BeEmpty())

			By("By creating a new BatchJob")
			ctx := context.Background()
			var (
				ImageName                = "gcr.io/spark-operator/spark:v3.1.1"
				ImagePullPolicy          = "Always"
				MainClass                = "org.apache.spark.examples.SparkPi"
				MainClassApplicationFile = "local:///opt/spark/examples/jars/spark-examples_2.12-3.1.1.jar"
			)

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
					},
				},
			}
			Expect(k8sClient.Create(ctx, batchJob)).Should(Succeed())

			By("Inspecting the Queue using the HTTP Handler")
			Eventually(func() ([]JobDescription, error) {
				req, err := http.NewRequest("GET", "/health-check", nil)
				Expect(err).NotTo(HaveOccurred())
				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(WS.GetQueue)
				handler.ServeHTTP(rr, req)

				// Check the status code is what we expect.
				Expect(rr.Code).Should(BeEquivalentTo(http.StatusOK))

				// Check the response body is what we expect.
				Expect(rr.Body.String()).ShouldNot(BeEmpty())

				var array []JobDescription
				err = json.Unmarshal(rr.Body.Bytes(), &array)

				return array, err
			}, timeout, interval).Should(
				And(
					WithTransform(func(p []JobDescription) int { return len(p) },
						BeIdenticalTo(1)),
					WithTransform(func(p []JobDescription) string { return p[0].JobName.Name },
						Equal(BatchJob)),
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
			Eventually(func() (batchjobv1alpha1.ApplicationStateType, error) {
				err := k8sClient.Get(ctx, batchjobLookupKey, createdBatchJob)
				if err != nil {
					return batchjobv1alpha1.FailedSubmissionState, err
				}
				return createdBatchJob.Status.State, nil
			}, timeout, interval).Should(BeEquivalentTo(batchjobv1alpha1.InQueueState))
		})

		It("Should release the Job from the Queue", func() {
			By("Getting Node Infos")
			Eventually(func() (map[string][]string, error) {
				req, err := http.NewRequest("GET", "/nodes", nil)
				Expect(err).NotTo(HaveOccurred())
				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(WS.GetNodes)
				handler.ServeHTTP(rr, req)

				// Check the status code is what we expect.
				Expect(rr.Code).Should(BeEquivalentTo(http.StatusOK))
				Expect(rr.Body.String()).ShouldNot(BeEmpty())

				var nodeMap = map[string][]string{}
				err = json.Unmarshal(rr.Body.Bytes(), &nodeMap)

				return nodeMap, err
			}, timeout, interval).
				Should(HaveKeyWithValue(TestNode, BeEmpty()))

			By("Verifying that the Queue contains one BatchJob")
			Eventually(func() ([]JobDescription, error) {
				req, err := http.NewRequest("GET", "/queue", nil)
				Expect(err).NotTo(HaveOccurred())
				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(WS.GetQueue)
				handler.ServeHTTP(rr, req)

				// Check the status code is what we expect.
				Expect(rr.Code).Should(BeEquivalentTo(http.StatusOK))

				// Check the response body is what we expect.
				Expect(rr.Body.String()).ShouldNot(BeEmpty())

				var array []JobDescription
				err = json.Unmarshal(rr.Body.Bytes(), &array)

				return array, err
			}, timeout, interval).Should(
				And(
					WithTransform(func(p []JobDescription) int { return len(p) },
						BeIdenticalTo(1)),
					WithTransform(func(p []JobDescription) string { return p[0].JobName.Name },
						Equal(BatchJob)),
				))

			By("Submitting a SchedulingDecision")
			var desiredScheduling = make(map[string][]types.NamespacedName)
			desiredScheduling[TestNode] = []types.NamespacedName{{Name: BatchJob, Namespace: BatchJobNamespace}}
			payloadBuf := new(bytes.Buffer)
			err := json.NewEncoder(payloadBuf).Encode(desiredScheduling)
			Expect(err).NotTo(HaveOccurred())

			func() {
				req, err := http.NewRequest("POST", "/schedule", payloadBuf)
				Expect(err).NotTo(HaveOccurred())
				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(WS.SubmitSchedule)
				handler.ServeHTTP(rr, req)

				// Check the status code is what we expect.
				Expect(rr.Code).Should(BeEquivalentTo(http.StatusOK))

				// Check the response body is what we expect.
				Expect(rr.Body.String()).ShouldNot(BeEmpty())

				var array []JobDescription
				err = json.Unmarshal(rr.Body.Bytes(), &array)
			}()

			By("Checking that a new SparkApplication was Created")
			Eventually(func() (*v1beta2.SparkApplication, error) {
				var sparkApp = &v1beta2.SparkApplication{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: BatchJob, Namespace: BatchJobNamespace}, sparkApp)
				return sparkApp, err
			}).Should(And(
				Not(BeNil()),
				WithTransform(func(sparkApp *v1beta2.SparkApplication) map[string]string {
					return sparkApp.Spec.Driver.SparkPodSpec.Annotations
				}, HaveKeyWithValue("external-scheduling-desired-node", TestNode)),
				WithTransform(func(sparkApp *v1beta2.SparkApplication) map[string]string {
					return sparkApp.Spec.Executor.SparkPodSpec.Annotations
				}, HaveKeyWithValue("external-scheduling-desired-node", TestNode)),
			))

			By("Checking that the Queue is now Empty")
			Eventually(func() ([]JobDescription, error) {
				req, err := http.NewRequest("GET", "/queue", nil)
				Expect(err).NotTo(HaveOccurred())
				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(WS.GetQueue)
				handler.ServeHTTP(rr, req)

				// Check the status code is what we expect.
				Expect(rr.Code).Should(BeEquivalentTo(http.StatusOK))

				// Check the response body is what we expect.
				Expect(rr.Body.String()).ShouldNot(BeEmpty())

				var array []JobDescription
				err = json.Unmarshal(rr.Body.Bytes(), &array)

				return array, err
			}, timeout, interval).Should(BeEmpty())

			By("By checking the BatchJob is now Starting")
			Eventually(func() (*batchjobv1alpha1.SimpleStatus, error) {
				batchjobLookupKey := types.NamespacedName{Name: BatchJob, Namespace: BatchJobNamespace}
				createdBatchJob := batchjobv1alpha1.Simple{}
				err := k8sClient.Get(ctx, batchjobLookupKey, &createdBatchJob)
				if err != nil {
					return nil, err
				}
				return &createdBatchJob.Status, nil
			}, timeout, interval).Should(
				WithTransform(func(status *batchjobv1alpha1.SimpleStatus) batchjobv1alpha1.ApplicationStateType {
					return status.State
				}, BeEquivalentTo(batchjobv1alpha1.SubmittedState)),
			)

			By("By Mocking the SparkOperator")
			var sparkApp = &v1beta2.SparkApplication{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: BatchJob, Namespace: BatchJobNamespace}, sparkApp)
			Expect(err).ToNot(HaveOccurred())

			sparkApp.Status.AppState.State = v1beta2.SubmittedState
			err = k8sClient.Status().Update(ctx, sparkApp)
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(1000)

			sparkApp.Status.AppState.State = v1beta2.RunningState
			err = k8sClient.Status().Update(ctx, sparkApp)
			Expect(err).ToNot(HaveOccurred())

			By("By checking the BatchJob is now Running")
			Eventually(func() (*batchjobv1alpha1.SimpleStatus, error) {
				batchjobLookupKey := types.NamespacedName{Name: BatchJob, Namespace: BatchJobNamespace}
				createdBatchJob := batchjobv1alpha1.Simple{}
				err := k8sClient.Get(ctx, batchjobLookupKey, &createdBatchJob)
				if err != nil {
					return nil, err
				}
				return &createdBatchJob.Status, nil
			}, timeout, interval).Should(
				WithTransform(func(status *batchjobv1alpha1.SimpleStatus) batchjobv1alpha1.ApplicationStateType {
					return status.State
				}, BeEquivalentTo(batchjobv1alpha1.RunningState)),
			)

		})
	})
})
