package controllers

import (
	. "context"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	. "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

type SparkController struct {
	client.Client
	Applications map[NamespacedName]*v1beta2.SparkApplication
}

func NewSparkController(client client.Client) *SparkController {
	return &SparkController{Client: client, Applications: map[NamespacedName]*v1beta2.SparkApplication{}}
}

const (
	SparkNoChange = iota
	SparkCreated
	SparkSubmitted
	SparkRunning
	SparkRemoved
	SparkProblem
)

func (sc *SparkController) hasSparkChanged(
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

		if CompareResourceVersion(ctx, old.ResourceVersion, spark.ResourceVersion) {
			if cmp.Equal(old.Status, spark.Status) {
				logger.Error(errors.New("no status change"), "SparkApplication has no status change")
				return SparkNoChange
			}
			return toTransitionEnumSpark(ctx, old.Status.AppState.State, spark.Status.AppState.State)
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

	ctrllog.FromContext(ctx).Info("Spark is in an unknown/unhandled state", "before", before, "after", after)

	return SparkProblem
}

func (sc SparkController) getLinkedSpark(context Context, name NamespacedName) (error, *v1beta2.SparkApplication) {
	var spark = &v1beta2.SparkApplication{}
	ctrllog.FromContext(context).V(TRACE).Info("Get SparkApplication")
	var err = sc.Client.Get(context, name, spark)

	if err != nil && k8serrors.IsNotFound(err) {
		ctrllog.FromContext(context).Info("Spark Application does not Exist")
		return nil, nil
	}

	return err, spark
}

func (sc *SparkController) manageSpark(ctx Context, nn NamespacedName, spark *v1beta2.SparkApplication) {

	if old, ok := sc.Applications[nn]; ok {
		if spark == nil {
			ctrllog.FromContext(ctx).Info("SparkApplication was removed", "old", "old")
			delete(sc.Applications, nn)
			return
		}

		if CompareResourceVersion(ctx, old.ResourceVersion, spark.ResourceVersion) {
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
