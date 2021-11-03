package controllers

import (
	"context"
	sparkv1beta2 "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/ls-1801/batchjob-operator/api/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

type ScheduleExecutor struct {
	SchedulingDecision map[*v1.Node][]*v1alpha1.Simple
}

func findNode(nodeName string, list v1.NodeList) *v1.Node {
	for _, node := range list.Items {
		if node.Name == nodeName {
			return &node
		}
	}

	return nil
}

func findJob(nn types.NamespacedName, list v1alpha1.SimpleList) *v1alpha1.Simple {
	for _, job := range list.Items {
		if job.Name == nn.Name && job.Namespace == nn.Namespace {
			return &job
		}
	}
	return nil
}

func NewScheduleExecutor(
	ctx context.Context,
	client *SimpleReconciler,
	desired map[string][]types.NamespacedName,
) (error, *ScheduleExecutor) {

	var nodeList = v1.NodeList{}
	if err := client.Client.List(ctx, &nodeList); err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could not List Nodes in the Cluster")
	}

	var jobList = v1alpha1.SimpleList{}
	if err := client.Client.List(ctx, &jobList); err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could not List Jobs in the Cluster")
	}

	var concrete = map[*v1.Node][]*v1alpha1.Simple{}

	for nodeName, jobNames := range desired {
		var cNode = findNode(nodeName, nodeList)
		if cNode == nil {
			return errors.New("node does not exist: " + nodeName), nil
		}

		var concreteJobs = []*v1alpha1.Simple{}
		for _, jobName := range jobNames {
			var cJob = findJob(jobName, jobList)
			if cJob == nil {
				return errors.New("job does not exist: " + jobName.String()), nil
			}
			concreteJobs = append(concreteJobs, cJob)
		}
		concrete[cNode] = concreteJobs
	}

	return nil, &ScheduleExecutor{SchedulingDecision: concrete}
}

func (se *ScheduleExecutor) Execute(ctx context.Context, client *SimpleReconciler) (error,
	map[string][]*sparkv1beta2.SparkApplication) {
	logger := ctrllog.FromContext(ctx)

	logger.Info("Applying scheduling decisions")

	var sparkApplications = map[string][]*sparkv1beta2.SparkApplication{}

	for node, jobs := range se.SchedulingDecision {
		for _, job := range jobs {
			err, sparkApp := se.scheduleJobOnNode(ctx, client, job, node)
			if err != nil {
				return err, nil
			}

			sparkApplications[node.Name] = append(sparkApplications[node.Name], sparkApp)
		}
	}

	return nil, sparkApplications

}

func (se *ScheduleExecutor) scheduleJobOnNode(ctx context.Context, client *SimpleReconciler, job *v1alpha1.Simple,
	node *v1.Node) (error,
	*sparkv1beta2.SparkApplication) {
	ctrllog.FromContext(ctx).Info("Creating SparkApplication from Job", "job", job)
	var spark = &sparkv1beta2.SparkApplication{}
	spark.Name = job.Name
	spark.Namespace = "default"
	spark.Spec = job.Spec.Spec

	err := ctrl.SetControllerReference(job, spark, client.Scheme)
	if err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could not set ControllerReference")
		return err, nil

	}
	annotationMap := make(map[string]string)
	annotationMap["external-scheduling-desired-node"] = node.Name

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

	if err := client.Create(ctx, spark); err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could Not Create SparkApplication")
		return err, nil
	}

	return nil, spark
}
