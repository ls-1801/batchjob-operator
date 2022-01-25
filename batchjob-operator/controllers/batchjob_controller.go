package controllers

import (
	. "context"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/google/go-cmp/cmp"
	. "github.com/ls-1801/batchjob-operator/api/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

type BatchJobController struct {
	client.Client
	ManagedJobs map[NamespacedName]*BatchJob
}

func NewBatchJobController(client client.Client) *BatchJobController {
	return &BatchJobController{Client: client, ManagedJobs: map[NamespacedName]*BatchJob{}}
}

const (
	NewJob = iota
	Removed
	Submitted
	Running
	Completed
	InQueue
	NoChange
	UnknownChange
)

func (c *BatchJobController) manageJob(ctx Context, batchJob *BatchJob, name NamespacedName) {
	if old, ok := c.ManagedJobs[name]; !ok {
		if batchJob == nil {
			return
		}

		ctrllog.FromContext(ctx).Info("Manage new Job")
	} else {
		if batchJob == nil {
			ctrllog.FromContext(ctx).Info("Managed Job was removed", "old", old)
			delete(c.ManagedJobs, name)
			return
		}

		if DidResourceVersionChange(ctx, old.ResourceVersion, batchJob.ResourceVersion) {
			ctrllog.FromContext(ctx).Info("Managed Job changes from Version " +
				old.ResourceVersion + " to " + batchJob.ResourceVersion)
		} else {
			ctrllog.FromContext(ctx).Info("Managed Job versions did not change from: " + old.ResourceVersion)
		}
	}

	c.ManagedJobs[name] = batchJob
}

func (c *BatchJobController) GetBatchJobChange(ctx Context, name NamespacedName, job *BatchJob) int {
	defer func() { c.manageJob(ctx, job, name) }()

	if old, ok := c.ManagedJobs[name]; ok {

		// Job was nil -> Job no longer in the Cluster
		if job == nil {
			ctrllog.FromContext(ctx).Info("Job was removed")
			return Removed
		}

		ctrllog.FromContext(ctx).Info("Check for differences", "diff", cmp.Diff(old, job))
		if DidResourceVersionChange(ctx, old.ResourceVersion, job.ResourceVersion) {
			if !cmp.Equal(old.Status, job.Status) {
				return toTransitionEnum(ctx, old.Status.State, job.Status.State)
			} else {
				return UnknownChange
			}
		}

		return NoChange

	}

	//This is the case if a BatchJob was deleted and the linked SparkApplication was also deleted
	if job == nil {
		return NoChange
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

	if before == RunningState && after == CompletedState {
		return Completed
	}

	ctrllog.FromContext(ctx).Info("Can't figure out transition!", "before", before, "after", after)
	return UnknownChange
}

func (c *BatchJobController) getBatchJob(context Context, name NamespacedName) (error, *BatchJob) {
	var batchJob = &BatchJob{}
	ctrllog.FromContext(context).V(TRACE).Info("Get BatchJob", "Namespaced-Name", name)
	var err = c.Client.Get(context, name, batchJob)

	if err != nil && k8serrors.IsNotFound(err) {
		ctrllog.FromContext(context).Info("BatchJob resource not found. Ignoring since object must be deleted")
		return nil, nil
	}

	return err, batchJob
}

func (c *BatchJobController) UpdateJobStatus(context Context, job *BatchJob, state ApplicationStateType) error {
	if job.Status.State == state {
		ctrllog.FromContext(context).Info("Job already in desired state", "New-State", state)
		return nil
	}

	var copiedJob = job.DeepCopy()
	copiedJob.Status.State = state
	ctrllog.FromContext(context).Info("Updating Status of Job", "job", copiedJob, "New-State", state)
	if err := c.Client.Status().Update(context, copiedJob); err != nil {
		ctrllog.FromContext(context).Error(err, "Failed to update BatchJob Status")
		return err
	}
	return nil
}

func (c *BatchJobController) ResumeBatchJob(ctx Context, job *BatchJob, spark *v1beta2.SparkApplication) error {
	ctrllog.FromContext(ctx).Info("Resuming BatchJob from already existing SparkApplication")
	//TODO: Figure out how to react here. Maybe copying the SparkStatus would be an idea.

	return nil
}

func (c *BatchJobController) AddStartEvent(ctx Context, job *BatchJob) error {
	var now = metav1.NewTime(time.Now())
	job.Status.ScheduleEvents = append(job.Status.ScheduleEvents, &BatchJobScheduledEvent{
		StartTimestamp:  &now,
		FinishTimestamp: nil,
		Successful:      nil,
		ScheduledNode:   "TODO",
	})

	if err := c.Client.Status().Update(ctx, job); err != nil {
		ctrllog.FromContext(ctx).Error(err, "Failed to update BatchJob Status")
		return err
	}
	return nil
}

func (c *BatchJobController) AddStopEvent(ctx Context, job *BatchJob, successful bool) error {
	var now = metav1.NewTime(time.Now())

	for _, event := range job.Status.ScheduleEvents {
		if event.FinishTimestamp != nil {
			continue
		}

		event.FinishTimestamp = &now
		event.Successful = &successful
	}

	if err := c.Client.Status().Update(ctx, job); err != nil {
		ctrllog.FromContext(ctx).Error(err, "Failed to update BatchJob Status")
		return err
	}

	return nil
}
