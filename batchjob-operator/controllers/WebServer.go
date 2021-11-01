package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	sparkv1beta2 "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	batchjobv1alpha1 "github.com/ls-1801/batchjob-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

type WebServer struct {
	Client   *SimpleReconciler
	JobQueue *JobQueue
}

func NewWebServer(client *SimpleReconciler) *WebServer {
	return &WebServer{Client: client, JobQueue: NewJobQueue()}
}

func (ws *WebServer) Start(context context.Context) error {
	var logger = ctrllog.FromContext(context)
	logger.Info("Init WebServer on port 9090")
	http.HandleFunc("/queue", ws.GetQueue)
	http.HandleFunc("/nodes", ws.GetNodes)
	logger.Info("Listening on port 9090")
	return http.ListenAndServe(":9090", nil)
}

func (ws *WebServer) SubmitJobToQueue(context context.Context, newJob types.NamespacedName) error {
	var logger = ctrllog.FromContext(context)

	logger.Info(
		"Adding new Job to the Queue",
	)

	ws.JobQueue.addJobToQueue(newJob)
	return nil
}

func (ws *WebServer) GetQueue(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.NotFound(w, req)
		return
	}

	var logger = ctrllog.FromContext(req.Context())
	defer logger.Info("Queue Request Done")
	logger.Info("Queue Request Started")

	var jobDescriptions = ws.JobQueue.copyQueue()

	w.Header().Set("Content-Type", "application/json")
	logger.Info("Current Queue contains", "queue", jobDescriptions)

	HandleError(json.NewEncoder(w).Encode(jobDescriptions))
}

func (ws *WebServer) GetNodes(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.NotFound(w, req)
		return
	}

	var logger = ctrllog.FromContext(req.Context())
	defer logger.Info("Node Request Done")
	logger.Info("Node Request Started")

	var nodeList = &corev1.NodeList{}
	if err := ws.Client.Client.List(req.Context(), nodeList); err != nil {
		logger.Error(err, "Error getting Nodes")
		HandleError1(fmt.Fprintln(w, "Error getting the Node list"))
	}

	var podList = &corev1.PodList{}
	if err := ws.Client.Client.List(req.Context(), podList); err != nil {
		logger.Error(err, "Error getting Pods")
		HandleError1(fmt.Fprintln(w, "Error getting the Pod list"))
	}

	var nodeMap = make(map[string][]string)

	for _, node := range nodeList.Items {
		if _, ok := nodeMap[node.Name]; !ok {
			nodeMap[node.Name] = []string{}
		}
	}

	for _, pod := range podList.Items {
		if val, ok := nodeMap[pod.Spec.NodeName]; ok {
			nodeMap[pod.Spec.NodeName] = append(val, pod.Name)
		} else {
			logger.Error(errors.New("Node not found: "+pod.Spec.NodeName), "Pod is Scheduled on an unknown Node")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	HandleError(json.NewEncoder(w).Encode(nodeMap))
}

func (ws *WebServer) SubmitSchedule(writer http.ResponseWriter, request *http.Request) {
	logger := ctrllog.FromContext(request.Context())

	if request.Method != http.MethodPost {
		http.NotFound(writer, request)
		return
	}

	var schedulingDecision = map[string][]types.NamespacedName{}
	err := json.NewDecoder(request.Body).Decode(&schedulingDecision)
	if err != nil {
		logger.Error(err, "When decoding Payload")
		http.Error(writer, "Could not decode JSON", http.StatusBadRequest)
		return
	}

	logger.Info("Applying scheduling decisions", "scheduling-decision", schedulingDecision)

	var responseMap = map[string][]*sparkv1beta2.SparkApplication{}

	for nodeName, jobsOnNode := range schedulingDecision {
		logger.Info("Jobs for Node", "node", nodeName, "jobs", jobsOnNode)
		for _, jobName := range jobsOnNode {
			logger.Info("Remove Job from queue", "job", jobName)
			var jobName = ws.JobQueue.removeFromQueue(jobName)
			if jobName == nil {
				http.Error(writer, "Could not find Job in Queue", http.StatusNotFound)
				return
			}

			logger.Info("Fetching Job from kubernetes", "job", jobName)
			var job = &batchjobv1alpha1.Simple{}
			err := ws.Client.Get(request.Context(), *jobName, job)
			if err != nil {
				http.Error(writer, "Could not find Job in Cluster", http.StatusNotFound)
				return
			}

			err, sparkApp := ws.ScheduleJobOnNode(request.Context(), job, nodeName)
			if err != nil {
				http.Error(writer, "Could not create SparkApplication for Job", http.StatusInternalServerError)
				logger.Error(err, "Failed create SparkApplication. Put BatchJob Back in the Queue", "BatchJob", jobName)
				ws.JobQueue.addJobToQueue(types.NamespacedName{
					Namespace: job.Namespace,
					Name:      job.Name,
				})
				return
			}

			responseMap[nodeName] = append(responseMap[nodeName], sparkApp)
		}
	}

	HandleError(json.NewEncoder(writer).Encode(responseMap))

}

func (ws *WebServer) ScheduleJobOnNode(ctx context.Context, job *batchjobv1alpha1.Simple, name string) (error, *sparkv1beta2.SparkApplication) {
	ctrllog.FromContext(ctx).Info("Creating SparkApplication from Job", "job", job)
	var spark = &sparkv1beta2.SparkApplication{}
	spark.Name = job.Name
	spark.Namespace = "default"
	spark.Spec = job.Spec.Spec

	err := ctrl.SetControllerReference(job, spark, ws.Client.Scheme)
	if err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could not set ControllerReference")
		return err, nil

	}
	annotationMap := make(map[string]string)
	annotationMap["external-scheduling-desired-node"] = name

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

	if err := ws.Client.Create(ctx, spark); err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could Not Create SparkApplication")
		return err, nil
	}

	return nil, spark
}
