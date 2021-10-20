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
	"fmt"
	"log"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	sparkv1beta2 "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	batchjobv1alpha1 "github.com/ls-1801/batchjob-operator/api/v1alpha1"
)

// SimpleReconciler reconciles a Simple object
type SimpleReconciler struct {
	client.Client
	Queue     *list.List
	Scheme    *runtime.Scheme
	Waiting   chan batchjobv1alpha1.Simple
	Scheduled chan batchjobv1alpha1.Simple
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
	logger.Info("Reconciler Loop Is Running")
	batchjob := &batchjobv1alpha1.Simple{}
	err := r.Get(ctx, req.NamespacedName, batchjob)
	logger.Info("Get Called")
	if err != nil {
		if errors.IsNotFound(err) {
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
	logger.Info("Found BatchJob", "BatchJob", batchjob)
	if batchjob.Status.InQueue {
		logger.Info("BatchJob is already in Queue no further Actions")
		return ctrl.Result{}, nil
	}

	// Check if the deployment already exists, if not create a new one
	found := &sparkv1beta2.SparkApplication{}
	// Maybe the spark application name should be used here
	err = r.Get(ctx, types.NamespacedName{Name: batchjob.Name, Namespace: batchjob.Namespace}, found)
	logger.Info("Get Spark")
	if err != nil && errors.IsNotFound(err) {
		logger.Info("New BatchJob found put in the queue")
		batchjob.Status.InQueue = true

		if err = r.Status().Update(ctx, batchjob); err != nil {
			logger.Error(err, "Failed to update BatchJob Status to InQueue", err)
			return ctrl.Result{}, err
		}

		logger.Info("BatchJob is Updated and put into the Queue", "BatchJob", batchjob.Status)

		r.Waiting <- *batchjob
		return ctrl.Result{}, nil

		// Define a new SparkApplication
		// dep := r.submitNewSparkApplication(batchjob)
		// logger.Info("Creating a new SparkApplication", "SparkApplication.Namespace", dep.Namespace, "SparkApplication.Name", dep.Name)
		// err = r.Create(ctx, dep)
		// if err != nil {
		// 	logger.Error(err, "Failed to create new SparkApplication", "SparkApplication.Namespace", dep.Namespace, "SparkApplication.Name", dep.Name)
		// 	return ctrl.Result{}, err
		// }
		// // SparkApplication created successfully
		// logger.Info("Status is now Running")
		// batchjob.Status.Running = true
		// if err = r.Update(ctx, batchjob); err != nil {
		// 	logger.Error(err, "Failed to update BatchJob Status to Running", err)
		// 	return ctrl.Result{}, err
		// }

	}
	// your logic here
	if err != nil {
		logger.Error(err, "Failed to get SparkApplications")
		return ctrl.Result{}, err
	}

	if found.Status.AppState.State == sparkv1beta2.CompletedState {
		batchjob.Status.Running = false
		// SparkApplication created successfully
		logger.Info("Status is no longer Running")
		batchjob.Status.Running = true
		if err = r.Update(ctx, batchjob); err != nil {
			logger.Error(err, "Failed to update BatchJob Status to no longer Running", err)
			return ctrl.Result{}, err
		}
	}

	logger.Info("NOT IMPLMENETED")

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
			logger.Info("Adding new Job to the Queue")
			ws.Client.Queue.PushBack(newJob)
		}
	}
}

func HandleError(err error) {
	log.Default().Print("HandleError:", err)
}

func HandleError1(i int, err error) {
	log.Default().Print("HandleError: ", "int", i, "error", err)
}

func (ws *WebServer) GetQueue(w http.ResponseWriter, req *http.Request) {
	var logger = ctrllog.FromContext(req.Context())
	defer logger.Info("Queue Request Done")
	logger.Info("Queue Request Started")
	logger.Info("Current Queue contains", "queue", ws.Client.Queue)
	w.Header().Set("Content-Type", "application/json")
	HandleError(json.NewEncoder(w).Encode(ws.Client.Queue))
}

func (ws *WebServer) GetNodes(w http.ResponseWriter, req *http.Request) {
	var logger = ctrllog.FromContext(req.Context())
	defer logger.Info("Node Request Done")
	logger.Info("Node Request Started")
	var podList = &corev1.PodList{}
	if err := ws.Client.Client.List(req.Context(), podList); err != nil {
		logger.Error(err, "Error getting Nodes")
		HandleError1(fmt.Fprintln(w, "Error getting the list"))
	}

	var nodeMap = make(map[string][]string)

	for _, pod := range podList.Items {
		if _, ok := nodeMap[pod.Spec.NodeName]; !ok {
			nodeMap[pod.Spec.NodeName] = make([]string, 0)
		}

		if val, ok := nodeMap[pod.Spec.NodeName]; ok {
			nodeMap[pod.Spec.NodeName] = append(val, pod.Name)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	HandleError(json.NewEncoder(w).Encode(nodeMap))
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
		//Owns(&sparkv1beta2.SparkApplication{}).
		Complete(r)
}
