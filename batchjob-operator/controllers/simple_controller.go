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
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=batchjob.gcr.io,resources=simples,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batchjob.gcr.io,resources=simples/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batchjob.gcr.io,resources=simples/finalizers,verbs=update
//+kubebuilder:rbac:groups=sparkoperator.k8s.io,resources=sparkapplications,verbs=get;list;watch;create;update;patch;delete

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
	log := ctrllog.FromContext(ctx)
	log.Info("Reconciler Loop Is Running")
	batchjob := &batchjobv1alpha1.Simple{}
	err := r.Get(ctx, req.NamespacedName, batchjob)
	log.Info("Get Called")
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("BatchJob resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Info("Error Not NotFound")
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get BatchJob")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	found := &sparkv1beta2.SparkApplication{}
	// Maybe the spark application name should be used here
	err = r.Get(ctx, types.NamespacedName{Name: batchjob.Name, Namespace: batchjob.Namespace}, found)
	log.Info("Get Spark")
	if err != nil && errors.IsNotFound(err) {
		// Define a new SparkApplication
		dep := r.submitNewSparkApplication(batchjob)
		log.Info("Creating a new SparkApplication", "SparkApplication.Namespace", dep.Namespace, "SparkApplication.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new SparkApplication", "SparkApplication.Namespace", dep.Namespace, "SparkApplication.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// SparkApplication created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}
	// your logic here
	if err != nil {
		log.Error(err, "Failed to get SparkApplications")
		return ctrl.Result{}, err
	}

	log.Info("NOT IMPLMENETED")

	return ctrl.Result{}, nil
}

func (r *SimpleReconciler) submitNewSparkApplication(simple *batchjobv1alpha1.Simple) *sparkv1beta2.SparkApplication {
	var spark = &sparkv1beta2.SparkApplication{}
	spark.Name = simple.Name
	spark.Namespace = "default"
	spark.Spec = simple.Spec.Spec
	return spark
}

// SetupWithManager sets up the controller with the Manager.
func (r *SimpleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mgr.GetLogger().Info("Setup")
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchjobv1alpha1.Simple{}).
		//Owns(&sparkv1beta2.SparkApplication{}).
		Complete(r)
}
