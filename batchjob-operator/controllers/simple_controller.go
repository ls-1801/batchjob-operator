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
	"errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	Scheme       *runtime.Scheme
	WebServer    *WebServer
	BatchJobCtrl *BatchJobController
	SparkCtrl    *SparkController
	JobQueue     *JobQueue
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
	logger.Info("================ LOOP TRIGGERED =============")

	var err, batchJob = r.BatchJobCtrl.getBatchJob(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Figure out what happened and why the loop was triggered
	switch r.BatchJobCtrl.hasChanged(ctx, req.NamespacedName, batchJob) {
	case Removed:
		logger.Info("BatchJob was removed. Missing Implementation!")
		//TODO: Delete associated SparkApplication
		return ctrl.Result{}, nil
	case NewJob:
		return r.handleNewJob(ctx, batchJob)
	case NoChange:
		// No change means that, the associated SparkApplication has changed
		err, spark := r.SparkCtrl.getLinkedSpark(ctx, req.NamespacedName)
		if err != nil {
			return ctrl.Result{}, err
		}
		switch r.SparkCtrl.hasSparkChanged(ctx, req.NamespacedName, spark) {
		case SparkNoChange:
			logger.Error(errors.New("unexpected loop trigger"), "Loop triggered without change")
			return ctrl.Result{}, nil
		case SparkCreated:
			logger.Info("SparkApplication Created")
			r.JobQueue.removeFromQueue(req.NamespacedName)
			return ctrl.Result{}, r.BatchJobCtrl.UpdateJobStatus(ctx, batchJob, batchjobv1alpha1.SubmittedState)
		case SparkSubmitted:
			logger.Info("SparkApp is now in Submitted State")
			return ctrl.Result{}, nil
		case SparkRunning:
			logger.Info("SparkApplication is Now Running")
			return ctrl.Result{}, r.BatchJobCtrl.UpdateJobStatus(ctx, batchJob, batchjobv1alpha1.RunningState)
		default:
			logger.Info("Spark did something")
			return ctrl.Result{}, errors.New("unexpected spark change")
		}
	case InQueue:
		logger.Info("Job is now in Queue", "job", req.NamespacedName)
	case Submitted:
		logger.Info("Job is now in Submitted, remove it from the Queue", "job", req.NamespacedName)
	case Running:
		logger.Info("Job is now in Running", "job", req.NamespacedName)
	default:
		logger.Error(errors.New("illegal state transition"), "Job Made in illegal State Transition")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
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

func (r *SimpleReconciler) handleNewJob(ctx context.Context, job *batchjobv1alpha1.Simple) (ctrl.Result, error) {

	sparkApplicationName := types.NamespacedName{Name: job.Name, Namespace: job.Namespace}
	err, spark := r.SparkCtrl.getLinkedSpark(ctx, sparkApplicationName)

	if err != nil {
		ctrllog.FromContext(ctx).Error(err, "Failed to get SparkApplications")
		return ctrl.Result{}, err
	}

	// Fringe case: Spark does already exist. We try to resume the BatchJob
	if spark != nil {
		return ctrl.Result{}, r.BatchJobCtrl.ResumeBatchJob(ctx, job, spark)
	}

	// Usual case: Spark does not exist. New Job is put into the Queue and its status is updated
	r.JobQueue.addJobToQueue(types.NamespacedName{
		Namespace: job.Namespace,
		Name:      job.Name,
	})

	if err = r.BatchJobCtrl.UpdateJobStatus(ctx, job, batchjobv1alpha1.InQueueState); err != nil {
		ctrllog.FromContext(ctx).Error(err, "Failed to update BatchJob Status to InQueue")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
