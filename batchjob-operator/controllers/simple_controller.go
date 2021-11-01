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
	"github.com/google/go-cmp/cmp"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"strconv"

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
	Scheme      *runtime.Scheme
	WebServer   *WebServer
	ManagedJobs map[types.NamespacedName]*batchjobv1alpha1.Simple
	SparkCtrl   *SparkController
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

	logger.Info("================ LOOP TRIGGERED ============")

	var err, batchJob = r.getBatchJob(ctx, req.NamespacedName)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			delete(r.ManagedJobs, req.NamespacedName)
			r.WebServer.JobQueue.removeFromQueue(req.NamespacedName)
			logger.Info("BatchJob resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	switch r.hasChanged(ctx, batchJob) {
	case NewJob:
		return r.handleNewJob(ctx, batchJob)
	case NoChange:
		switch r.SparkCtrl.hasSparkChanged(ctx, batchJob) {
		case SparkNoChange:
			logger.Error(errors.New("unexpected loop trigger"), "Loop triggered without change")
			return ctrl.Result{}, nil
		case SparkCreated:
			logger.Info("SparkApplication Created")
			HandleError(r.UpdateJobStatus(ctx, batchJob, batchjobv1alpha1.SubmittedState))
		case SparkRunning:
			logger.Info("SparkApplication is Now Running")
			HandleError(r.UpdateJobStatus(ctx, batchJob, batchjobv1alpha1.RunningState))
		default:
			logger.Info("Spark did something")

		}
	case InQueue:
		logger.Info("Job is now in Queue", "job", req.NamespacedName)
	case Submitted:
		logger.Info("Job is now in Submitted", "job", req.NamespacedName)
	case Running:
		logger.Info("Job is now in Running", "job", req.NamespacedName)
	default:
		logger.Error(errors.New("illegal state transition"), "Job Made in illegal State Transition")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *SimpleReconciler) manageJob(ctx context.Context, batchJob *batchjobv1alpha1.Simple) {

	var nn = types.NamespacedName{Name: batchJob.Name, Namespace: batchJob.Namespace}
	if old, ok := r.ManagedJobs[nn]; ok {
		if r.compareResourceVersion(ctx, old.ResourceVersion, batchJob.ResourceVersion) {
			ctrllog.FromContext(ctx).Info("Managed Job changes from Version "+old.ResourceVersion+" to "+batchJob.ResourceVersion,
				"name", nn)
		} else {
			ctrllog.FromContext(ctx).Info("Managed Job versions did not change from: "+old.ResourceVersion,
				"name", nn)
		}
	} else {
		ctrllog.FromContext(ctx).Info("Manage new Job", "name", nn)
	}

	r.ManagedJobs[nn] = batchJob
}

const (
	NewJob = iota
	Submitted
	Running
	InQueue
	NoChange
	UnknownChange
)

func (r *SimpleReconciler) compareResourceVersion(ctx context.Context, rv1 string, rv2 string) bool {
	oldRV, err := strconv.Atoi(rv1)
	if err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could not extract old Resource Version")
	}
	newRV, err := strconv.Atoi(rv2)
	if err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could not extract new Resource Version")
	}

	return oldRV < newRV
}

func (r *SimpleReconciler) hasChanged(ctx context.Context, job *batchjobv1alpha1.Simple) int {
	defer func() { r.manageJob(ctx, job) }()
	if old, ok := r.ManagedJobs[types.NamespacedName{
		Name:      job.Name,
		Namespace: job.Namespace,
	}]; ok {
		ctrllog.FromContext(ctx).Info("Check for differences", "old", *old, "new", *job)

		if r.compareResourceVersion(ctx, old.ResourceVersion, job.ResourceVersion) {
			if !cmp.Equal(old.Status, job.Status) {
				return toTransitionEnum(ctx, old.Status.State, job.Status.State)
			} else {
				return UnknownChange
			}
		}

		return NoChange

	}

	ctrllog.FromContext(ctx).Info("Job is new", "new", types.NamespacedName{
		Name:      job.Name,
		Namespace: job.Namespace,
	})
	return NewJob
}

func toTransitionEnum(ctx context.Context, before batchjobv1alpha1.ApplicationStateType, after batchjobv1alpha1.ApplicationStateType) int {
	if before == batchjobv1alpha1.NewState && after == batchjobv1alpha1.InQueueState {
		return InQueue
	}

	if before == batchjobv1alpha1.InQueueState && after == batchjobv1alpha1.SubmittedState {
		return Submitted
	}

	if before == batchjobv1alpha1.SubmittedState && after == batchjobv1alpha1.RunningState {
		return Running
	}

	ctrllog.FromContext(ctx).Info("Can't figure out transition!", "before", before, "after", after)
	return UnknownChange
}

func (r *SimpleReconciler) getBatchJob(context context.Context, name types.NamespacedName) (error, *batchjobv1alpha1.Simple) {
	var batchJob = &batchjobv1alpha1.Simple{}
	ctrllog.FromContext(context).V(TRACE).Info("Get BatchJob", "Namespaced-Name", name)
	var err = r.Get(context, name, batchJob)
	return err, batchJob
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

func (r *SimpleReconciler) handleNewJob(ctx context.Context, job *batchjobv1alpha1.Simple) (ctrl.Result, error) {
	r.manageJob(ctx, job)
	sparkApplicationName := types.NamespacedName{Name: job.Name, Namespace: job.Namespace}
	ctrllog.FromContext(ctx).V(TRACE).Info("Find Existing Spark Application", "Namespaced Name", sparkApplicationName)
	// Check if the SparkApplication already exists, if not create a new one
	found := &sparkv1beta2.SparkApplication{}
	// Maybe the spark application name should be used here
	err := r.Get(ctx, sparkApplicationName, found)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			ctrllog.FromContext(ctx).V(TRACE).Info("SparkApplication was not found in Cluster")
			err := r.WebServer.SubmitJobToQueue(ctx, types.NamespacedName{
				Namespace: job.Namespace,
				Name:      job.Name,
			})

			if err != nil {
				ctrllog.FromContext(ctx).Error(err, "Failed to add the Job to the JobQueue")
				return ctrl.Result{}, err
			}

			err = r.UpdateJobStatus(ctx, job, batchjobv1alpha1.InQueueState)
			if err != nil {
				ctrllog.FromContext(ctx).Error(err, "Failed to update BatchJob Status to InQueue")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil

		}

		ctrllog.FromContext(ctx).Error(err, "Failed to get SparkApplications")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SimpleReconciler) UpdateJobStatus(context context.Context, job *batchjobv1alpha1.Simple, state batchjobv1alpha1.ApplicationStateType) error {
	var copiedJob = job.DeepCopy()
	copiedJob.Status.State = state
	ctrllog.FromContext(context).Info("Updating Status of Job", "job", copiedJob, "New-State", state)
	if err := r.Status().Update(context, copiedJob); err != nil {
		ctrllog.FromContext(context).Error(err, "Failed to update BatchJob Status", err)
		return err
	}
	return nil
}
