package controllers

import (
	. "context"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	. "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

const SparkRoleLabel = "spark-role"
const SparkOperatorAppNameLabel = "sparkoperator.k8s.io/app-name"

type SparkController struct {
	client.Client
	Applications map[NamespacedName]*v1beta2.SparkApplication
	Pods         map[NamespacedName]*SparkControlledPods
}

type SparkControlledPods struct {
	Driver    *corev1.Pod
	Executors []*corev1.Pod
}

func (scp *SparkControlledPods) didChange(ctx Context, old *SparkControlledPods) []string {
	var log = ctrllog.FromContext(WithValue(ctx, "SparkControlledPods", scp))
	var changes []string
	defer func() { log.Info("Changes", "changes", changes) }()

	if old == nil {
		changes = append(changes, "Spark Pod was created")
		return changes
	}

	if scp.Driver != nil || old.Driver == nil {
		changes = append(changes, "Driver does now exist")
	} else if scp.Driver == nil || old.Driver != nil {
		changes = append(changes, "Driver was removed")
	} else if DidResourceVersionChange(ctx, old.Driver.ResourceVersion, scp.Driver.ResourceVersion) {
		changes = append(changes, "Driver resource version changed")
	}

	for _, newExecutor := range scp.Executors {
		var oldExecutor *corev1.Pod
		for _, executor := range old.Executors {
			if newExecutor.UID == executor.UID {
				oldExecutor = executor
			}
		}

		if newExecutor != nil || oldExecutor == nil {
			changes = append(changes, "Executor does now exist")
		} else if DidResourceVersionChange(ctx, oldExecutor.ResourceVersion, newExecutor.ResourceVersion) {
			changes = append(changes, "Executor resource version changed")
		}
	}

	for _, oldExecutor := range old.Executors {
		var newExecutor *corev1.Pod
		for _, executor := range scp.Executors {
			if oldExecutor.UID == executor.UID {
				newExecutor = executor
			}
		}

		if newExecutor == nil || oldExecutor != nil {
			changes = append(changes, "Executor was removed")
		}
	}

	return changes
}

func NewSparkController(client client.Client) *SparkController {
	return &SparkController{Client: client, Applications: map[NamespacedName]*v1beta2.SparkApplication{}, Pods: map[NamespacedName]*SparkControlledPods{}}
}

const (
	SparkNoChange = iota
	SparkLinkedPodChanged
	SparkExecutorChanges
	SparkDriverChanges
	SparkCreated
	SparkSubmitted
	SparkRunning
	SparkRemoved
	SparkProblem
	SparkSucceeding
	SparkCompleted
	SparkFailed
	UnhandledChange
)

func (sc *SparkController) GetSparkChange(
	ctx Context,
	name NamespacedName,
	spark *v1beta2.SparkApplication,
) int {
	var logger = ctrllog.FromContext(ctx)
	defer func() { sc.manageSpark(ctx, name, spark) }()

	if old, ok := sc.Applications[name]; ok {
		if spark == nil {
			logger.Info("SparkApplication was removed")
			return SparkRemoved
		}

		// Check Resource Version
		if DidResourceVersionChange(ctx, old.ResourceVersion, spark.ResourceVersion) {
			if cmp.Equal(old.Status, spark.Status) {
				logger.Error(errors.New("no status change"), "SparkApplication has no status change")
				return SparkNoChange
			}

			if old.Status.AppState.State != spark.Status.AppState.State {
				return toTransitionEnumSpark(ctx, old.Status.AppState.State, spark.Status.AppState.State)
			}

			// Changes that are caused by the SparkApps pods will occur twice, once for the SparkApp and once for the
			// Linked pods
			if !cmp.Equal(old.Status.ExecutorState, spark.Status.ExecutorState) {
				logger.Info("ExecutorState has changed", "diff", cmp.Diff(old.Status.ExecutorState, spark.Status.ExecutorState))
				return SparkExecutorChanges
			}

			if !cmp.Equal(old.Status.DriverInfo, spark.Status.DriverInfo) {
				logger.Info("DriverInfo has changed", "diff", cmp.Diff(old.Status.DriverInfo, spark.Status.DriverInfo))
				return SparkDriverChanges
			}

			logger.Info("Unhandled Spark Status change", "diff", cmp.Diff(old.Status, spark.Status))
			return UnhandledChange
		}

		// Check Linked Pods
		err, podChanges := sc.GetLinkedSparkPodChange(ctx, name)
		if err != nil {
			logger.Error(err, "Could not detect Spark Change")
			return SparkNoChange
		}

		if podChanges {
			return SparkLinkedPodChanged
		}

		return SparkNoChange
	}

	if spark == nil {
		logger.Info("SparkApplication, was never managed nor does it exist")
		return SparkNoChange
	}

	return SparkCreated
}

func toTransitionEnumSpark(ctx Context, before v1beta2.ApplicationStateType, after v1beta2.ApplicationStateType) int {

	if before == v1beta2.NewState && after == v1beta2.SubmittedState {
		ctrllog.FromContext(ctx).Info("Spark went to Submitted State")
		return SparkSubmitted
	}

	if (before == v1beta2.NewState || before == v1beta2.SubmittedState) && after == v1beta2.RunningState {
		ctrllog.FromContext(ctx).Info("Spark went to Running State")
		return SparkRunning
	}

	if before == v1beta2.RunningState && after == v1beta2.SucceedingState {
		ctrllog.FromContext(ctx).Info("Spark went to Succeeding State")
		return SparkSucceeding
	}

	if before == v1beta2.SucceedingState && after == v1beta2.CompletedState {
		ctrllog.FromContext(ctx).Info("Spark went to Completed State")
		return SparkCompleted
	}

	if after == v1beta2.FailedState || after == v1beta2.FailingState {
		ctrllog.FromContext(ctx).Info("Spark Job Failed")
		return SparkFailed
	}

	ctrllog.FromContext(ctx).Info("Spark is in an unknown/unhandled state", "before", before, "after", after)

	return SparkProblem
}

func (sc SparkController) LinkedSparkApplication(context Context, name NamespacedName) (error, *v1beta2.SparkApplication) {
	var spark = &v1beta2.SparkApplication{}
	ctrllog.FromContext(context).V(TRACE).Info("Get SparkApplication")
	var err = sc.Client.Get(context, name, spark)

	if err != nil && k8serrors.IsNotFound(err) {
		ctrllog.FromContext(context).Info("Spark Application does not Exist")
		return nil, nil
	}

	return err, spark
}

func (sc SparkController) LinkedSparkApplicationPods(ctx Context, name NamespacedName) (error, *corev1.PodList) {
	var podList = &corev1.PodList{}
	ctrllog.FromContext(ctx).V(TRACE).Info("Get Pods linked to SparkApplication")

	err := sc.Client.List(
		ctx,
		podList,
		client.InNamespace(name.Namespace),
		client.MatchingLabels{SparkOperatorAppNameLabel: name.Name},
	)

	if err != nil {
		return err, nil
	}

	return nil, podList
}

func (sc SparkController) GetLinkedSparkPodChange(ctx Context, name NamespacedName) (error, bool) {
	err, pods := sc.LinkedSparkApplicationPods(ctx, name)
	if err != nil {
		return err, false
	}

	return nil, sc.manageSparkPods(ctx, name, pods)
}

func (sc SparkController) DeleteLinkedSpark(context Context, name NamespacedName) error {
	err, spark := sc.LinkedSparkApplication(context, name)
	if err != nil {
		return err
	}

	ctrllog.FromContext(context).Info("Delete SparkApplication")
	if spark == nil {
		ctrllog.FromContext(context).Info("Spark Application does not exist")
	}

	return sc.Client.Delete(context, spark)
}

func (sc *SparkController) manageSpark(ctx Context, nn NamespacedName, spark *v1beta2.SparkApplication) {

	if old, ok := sc.Applications[nn]; ok {
		if spark == nil {
			ctrllog.FromContext(ctx).Info("No longer keeping track of Spark", "old", old)
			delete(sc.Applications, nn)
			return
		}

		if DidResourceVersionChange(ctx, old.ResourceVersion, spark.ResourceVersion) {
			ctrllog.FromContext(ctx).Info("Managed SparkApplication changes from Version " +
				old.ResourceVersion + " to " + spark.ResourceVersion)
		} else {
			ctrllog.FromContext(ctx).Info("Managed SparkApplication versions did not change from: " + old.ResourceVersion)
		}
	} else {
		if spark == nil {
			ctrllog.FromContext(ctx).Info("SparkApplication was never managed")
			return
		}

		ctrllog.FromContext(ctx).Info("Manage new SparkApplication", "name", nn)
	}

	sc.Applications[nn] = spark
}

func toSparkControlledPods(pods *corev1.PodList) (error, *SparkControlledPods) {
	var sparkPods = &SparkControlledPods{
		Driver:    nil,
		Executors: nil,
	}

	for _, pod := range pods.Items {

		if role, ok := pod.Labels[SparkRoleLabel]; ok {
			if role == "driver" {
				sparkPods.Driver = &pod
			} else if role == "executor" {
				sparkPods.Executors = append(sparkPods.Executors, &pod)
			}
		} else {
			return errors.New("Pod without spark-role label"), nil
		}
	}

	return nil, sparkPods
}

func (sc *SparkController) manageSparkPods(ctx Context, nn NamespacedName, pods *corev1.PodList) bool {

	err, sparkPods := toSparkControlledPods(pods)
	if err != nil {
		ctrllog.FromContext(ctx).Error(err, "When inspecting SparkApplication Pods")
		return false
	}

	var changes = sparkPods.didChange(ctx, sc.Pods[nn])
	sc.Pods[nn] = sparkPods

	return len(changes) > 0
}
