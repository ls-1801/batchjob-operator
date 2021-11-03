package controllers

import (
	. "context"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/google/go-cmp/cmp"
	. "github.com/ls-1801/batchjob-operator/api/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	. "k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

type BatchJobController struct {
	Client      *SimpleReconciler
	ManagedJobs map[NamespacedName]*Simple
}

func NewBatchJobController(client *SimpleReconciler) *BatchJobController {
	return &BatchJobController{Client: client, ManagedJobs: map[NamespacedName]*Simple{}}
}

const (
	NewJob = iota
	Removed
	Submitted
	Running
	InQueue
	NoChange
	UnknownChange
)

func (c *BatchJobController) manageJob(ctx Context, batchJob *Simple, name NamespacedName) {
	if old, ok := c.ManagedJobs[name]; ok {
		if batchJob == nil {
			ctrllog.FromContext(ctx).Info("Managed Job was removed", "old", old)
			delete(c.ManagedJobs, name)
			return
		}

		if CompareResourceVersion(ctx, old.ResourceVersion, batchJob.ResourceVersion) {
			ctrllog.FromContext(ctx).Info("Managed Job changes from Version " +
				old.ResourceVersion + " to " + batchJob.ResourceVersion)
		} else {
			ctrllog.FromContext(ctx).Info("Managed Job versions did not change from: " + old.ResourceVersion)
		}
	} else {
		if batchJob == nil {
			ctrllog.FromContext(ctx).Info("Job was never managed")
			return
		}

		ctrllog.FromContext(ctx).Info("Manage new Job")
	}

	c.ManagedJobs[name] = batchJob
}

func (c *BatchJobController) hasChanged(ctx Context, name NamespacedName, job *Simple) int {
	defer func() { c.manageJob(ctx, job, name) }()

	if old, ok := c.ManagedJobs[name]; ok {

		// Job was nil -> Job no longer in the Cluster
		if job == nil {
			ctrllog.FromContext(ctx).Info("Job was removed")
			return Removed
		}

		ctrllog.FromContext(ctx).Info("Check for differences", "old", *old, "new", *job)
		if CompareResourceVersion(ctx, old.ResourceVersion, job.ResourceVersion) {
			if !cmp.Equal(old.Status, job.Status) {
				return toTransitionEnum(ctx, old.Status.State, job.Status.State)
			} else {
				return UnknownChange
			}
		}

		return NoChange

	}

	//Unexpected Case where an unmanaged Job gets removed
	//This Could happen if the Loop was triggered by an SparkApplication that was controlled by a deleted Job.
	//In any case we should remove the SparkApplication
	if job == nil {
		ctrllog.FromContext(ctx).Info("Job was removed, even though it was never managed")
		return Removed
	}

	ctrllog.FromContext(ctx).Info("Job is new", "new", NamespacedName{
		Name:      job.Name,
		Namespace: job.Namespace,
	})
	return NewJob
}

func toTransitionEnum(ctx Context, before ApplicationStateType, after ApplicationStateType) int {
	if before == NewState && after == InQueueState {
		return InQueue
	}

	if before == InQueueState && after == SubmittedState {
		return Submitted
	}

	if before == SubmittedState && after == RunningState {
		return Running
	}

	ctrllog.FromContext(ctx).Info("Can't figure out transition!", "before", before, "after", after)
	return UnknownChange
}

func (c *BatchJobController) getBatchJob(context Context, name NamespacedName) (error, *Simple) {
	var batchJob = &Simple{}
	ctrllog.FromContext(context).V(TRACE).Info("Get BatchJob", "Namespaced-Name", name)
	var err = c.Client.Get(context, name, batchJob)

	if err != nil && k8serrors.IsNotFound(err) {
		ctrllog.FromContext(context).Info("BatchJob resource not found. Ignoring since object must be deleted")
		delete(c.ManagedJobs, name)
		return nil, nil
	}

	return err, batchJob
}

func (c *BatchJobController) UpdateJobStatus(context Context, job *Simple, state ApplicationStateType) error {
	var copiedJob = job.DeepCopy()
	copiedJob.Status.State = state
	ctrllog.FromContext(context).Info("Updating Status of Job", "job", copiedJob, "New-State", state)
	if err := c.Client.Status().Update(context, copiedJob); err != nil {
		ctrllog.FromContext(context).Error(err, "Failed to update BatchJob Status", err)
		return err
	}
	return nil
}

func (c *BatchJobController) ResumeBatchJob(ctx Context, job *Simple, spark *v1beta2.SparkApplication) error {
	ctrllog.FromContext(ctx).Info("Resuming BatchJob from already existing SparkApplication")
	//TODO: Figure out how to react here. Maybe copying the SparkStatus would be an idea.

	return nil
}
