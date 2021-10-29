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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"log"

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
	Scheme    *runtime.Scheme
	WebServer *WebServer
}

type JobDescription struct {
	JobName types.NamespacedName
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
			return ctrl.Result{}, r.WebServer.SubmitJobToQueue(ctx, batchJob)
		}

		logger.Error(err, "Failed to get SparkApplications")
		return ctrl.Result{}, err
	}

	logger.Info("SparkApplication already Exists. No Further Actions")

	return ctrl.Result{}, nil
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

// SetupWithManager sets up the controller with the Manager.
func (r *SimpleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mgr.GetLogger().Info("Setup")
	err := mgr.Add(NewWebServer(r))
	if err != nil {
		mgr.GetLogger().Error(err, "Problem adding WebServer to Manager")
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchjobv1alpha1.Simple{}).
		Owns(&sparkv1beta2.SparkApplication{}).
		Complete(r)
}
