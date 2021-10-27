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
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	sparkv1beta2 "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	batchjobv1alpha1 "github.com/ls-1801/batchjob-operator/api/v1alpha1"
)

const TRACE = 100

// SimpleReconciler reconciles a Simple object
type SimpleReconciler struct {
	client.Client
	Queue     *list.List
	Scheme    *runtime.Scheme
	Waiting   chan *batchjobv1alpha1.Simple
	WebServer WebServer
}

type JobDescription struct {
	JobName string
}

type PodDescription struct {
	PodName  string
	NodeName string
}

//+kubebuilder:rbac:groups=batchjob.gcr.io,resources=simples,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batchjob.gcr.io,resources=simples/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batchjob.gcr.io,resources=simples/finalizers,verbs=update
//+kubebuilder:rbac:groups=sparkoperator.k8s.io,resources=sparkapplications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Simple object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *SimpleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrllog.FromContext(ctx)
	trace := logger.V(TRACE)

	trace.Info("Find existing BatchJob", "Namespaced Name", req.NamespacedName)
	batchJob := &batchjobv1alpha1.Simple{}
	err := r.Get(ctx, req.NamespacedName, batchJob)

	if err != nil {

		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Info("BatchJob resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		logger.Info("Error Not NotFound")
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get BatchJob")
		return ctrl.Result{}, err
	}

	logger.Info("Found BatchJob", "BatchJob", batchJob)

	if batchJob.Status.State == batchjobv1alpha1.RunningState ||
		batchJob.Status.State == batchjobv1alpha1.InQueueState ||
		batchJob.Status.State == batchjobv1alpha1.CompletedState {
		logger.Info("BatchJob is already in Queue or Running no further Actions")
		return ctrl.Result{}, nil
	}

	sparkApplicationName := types.NamespacedName{Name: batchJob.Name, Namespace: batchJob.Namespace}
	trace.Info("Find Existing Spark Application", "Namespaced Name", sparkApplicationName)
	// Check if the SparkApplication already exists, if not create a new one
	found := &sparkv1beta2.SparkApplication{}
	// Maybe the spark application name should be used here
	err = r.Get(ctx, sparkApplicationName, found)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			trace.Info("SparkApplication was not found in Cluster")
			r.Waiting <- batchJob
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Failed to get SparkApplications")
		return ctrl.Result{}, err
	}

	logger.Info("SparkApplication already Exists. No Further Actions")

	return ctrl.Result{}, nil
}

func (r *SimpleReconciler) submitNewSparkApplication(simple *batchjobv1alpha1.Simple) *sparkv1beta2.SparkApplication {
	var spark = &sparkv1beta2.SparkApplication{}
	spark.Name = simple.Name
	spark.Namespace = "default"
	spark.Spec = simple.Spec.Spec
	return spark
}

type WebServer struct {
	Client *SimpleReconciler
}

func (ws WebServer) Start(context context.Context) error {
	var logger = ctrllog.FromContext(context)
	logger.Info("Init WebServer on port 9090")
	http.HandleFunc("/queue", ws.GetQueue)
	http.HandleFunc("/nodes", ws.GetNodes)
	go ws.ListenForNewJobs(context)
	logger.Info("Listening on port 9090")
	return http.ListenAndServe(":9090", nil)
}

func (ws *WebServer) ListenForNewJobs(context context.Context) {
	var logger = ctrllog.FromContext(context)
	logger.Info("Waiting for New Jobs to be added to the Queue")
	for {
		select {
		case <-context.Done():
			return
		case newJob := <-ws.Client.Waiting:
			newJob.Status.InQueue = true
			if err := ws.Client.Status().Update(context, newJob); err != nil {
				logger.Error(err, "Failed to update BatchJob Status to InQueue", err)
				continue
			}

			logger.Info(
				"Adding new Job to the Queue",
				"Namespaced Name",
				types.NamespacedName{Name: newJob.Name, Namespace: newJob.Namespace},
			)

			ws.Client.Queue.PushFront(newJob)
		}
	}
}

func HandleError(err error) {
	if err == nil {
		return
	}
	log.Default().Print("HandleError:", err)
}

func HandleError1(i int, err error) {
	if err == nil {
		return
	}
	log.Default().Print("HandleError: ", "int", i, "error", err)
}

func (ws *WebServer) GetQueue(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.NotFound(w, req)
		return
	}

	var logger = ctrllog.FromContext(req.Context())
	defer logger.Info("Queue Request Done")
	logger.Info("Queue Request Started")

	var jobDescriptions = make([]JobDescription, ws.Client.Queue.Len())
	var idx = 0
	for e := ws.Client.Queue.Front(); e != nil; e = e.Next() {
		jobDescriptions[idx] = JobDescription{
			JobName: e.Value.(*batchjobv1alpha1.Simple).Name,
		}
		idx = idx + 1
	}

	w.Header().Set("Content-Type", "application/json")
	logger.Info("Current Queue contains", "queue", jobDescriptions)

	HandleError(json.NewEncoder(w).Encode(jobDescriptions))
}

func (ws *WebServer) GetNodes(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.NotFound(w, req)
		return
	}

	var logger = ctrllog.FromContext(req.Context())
	defer logger.Info("Node Request Done")
	logger.Info("Node Request Started")

	var nodeList = &corev1.NodeList{}
	if err := ws.Client.Client.List(req.Context(), nodeList); err != nil {
		logger.Error(err, "Error getting Nodes")
		HandleError1(fmt.Fprintln(w, "Error getting the Node list"))
	}

	var podList = &corev1.PodList{}
	if err := ws.Client.Client.List(req.Context(), podList); err != nil {
		logger.Error(err, "Error getting Pods")
		HandleError1(fmt.Fprintln(w, "Error getting the Pod list"))
	}

	var nodeMap = make(map[string][]string)

	for _, node := range nodeList.Items {
		if _, ok := nodeMap[node.Name]; !ok {
			nodeMap[node.Name] = []string{}
		}
	}

	for _, pod := range podList.Items {
		if val, ok := nodeMap[pod.Spec.NodeName]; ok {
			nodeMap[pod.Spec.NodeName] = append(val, pod.Name)
		} else {
			logger.Error(errors.New("Node not found: "+pod.Spec.NodeName), "Pod is Scheduled on an unknown Node")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	HandleError(json.NewEncoder(w).Encode(nodeMap))
}

func (ws WebServer) SubmitSchedule(writer http.ResponseWriter, request *http.Request) {
	logger := ctrllog.FromContext(request.Context())

	if request.Method != http.MethodPost {
		http.NotFound(writer, request)
		return
	}

	var schedulingDecision = map[string][]string{}
	err := json.NewDecoder(request.Body).Decode(&schedulingDecision)
	if err != nil {
		logger.Error(err, "When decoding Payload")
		http.Error(writer, "Could not decode JSON", http.StatusBadRequest)
	}

	var responseMap = map[string][]*sparkv1beta2.SparkApplication{}

	for nodeName, jobsOnNode := range schedulingDecision {
		for _, jobName := range jobsOnNode {
			var job = ws.RemoveFromQueue(jobName)
			if job == nil {
				http.Error(writer, "Could not find Job: "+jobName, http.StatusNotFound)
			}
			err, sparkApp := ws.ScheduleJobOnNode(request.Context(), job, nodeName)
			if err != nil {
				http.Error(writer, "Could not create SparkApplication for Job: "+jobName, http.StatusInternalServerError)
				logger.Error(err, "Failed create SparkApplication. Put BatchJob Back in the Queue", "BatchJob", job)
				ws.Client.Queue.PushFront(job)
			}

			job.Status.State = batchjobv1alpha1.SubmittedState

			if err := ws.Client.Status().Update(request.Context(), job); err != nil {
				logger.Error(err, "Failed to update BatchJob Status to Starting", err)
				http.Error(writer, "Failed to update BatchJob Status to Starting: "+jobName, http.StatusInternalServerError)
			}

			responseMap[nodeName] = append(responseMap[nodeName], sparkApp)
		}
	}

	HandleError(json.NewEncoder(writer).Encode(responseMap))

}

func (ws WebServer) RemoveFromQueue(jobName string) *batchjobv1alpha1.Simple {
	for e := ws.Client.Queue.Front(); e != nil; e = e.Next() {
		if e.Value.(*batchjobv1alpha1.Simple).Name == jobName {
			return ws.Client.Queue.Remove(e).(*batchjobv1alpha1.Simple)
		}
	}
	return nil
}

func (ws WebServer) ScheduleJobOnNode(ctx context.Context, job *batchjobv1alpha1.Simple, name string) (error, *sparkv1beta2.SparkApplication) {
	var spark = &sparkv1beta2.SparkApplication{}
	spark.Name = job.Name
	spark.Namespace = "default"
	spark.Spec = job.Spec.Spec

	annotationMap := make(map[string]string)
	annotationMap["external-scheduling-desired-node"] = name

	spark.Spec.Driver = sparkv1beta2.DriverSpec{
		SparkPodSpec: sparkv1beta2.SparkPodSpec{
			Annotations: annotationMap,
		},
	}

	spark.Spec.Executor = sparkv1beta2.ExecutorSpec{
		SparkPodSpec: sparkv1beta2.SparkPodSpec{
			Annotations: annotationMap,
		},
	}

	if err := ws.Client.Create(ctx, spark); err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could Not Create SparkApplication")
		return err, nil
	}

	return nil, spark
}

// SetupWithManager sets up the controller with the Manager.
func (r *SimpleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mgr.GetLogger().Info("Setup")
	err := mgr.Add(WebServer{Client: r})
	if err != nil {
		mgr.GetLogger().Error(err, "Problem adding WebServer to Manager")
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchjobv1alpha1.Simple{}).
		Owns(&sparkv1beta2.SparkApplication{}).
		Complete(r)
}
