A BatchJob in the context of the External-Scheduler-Interface is a wrapper around a Distributed Dataflow application. The current version of the Interface supports Spark and Flink. The BatchJob resource can also hold any data that an external may want to associate with the job. Listing \ref{lst:batchJobManifest} is an example configuration for the BatchJob resource. The configuration shown is only a shortened version. Complete examples are inside the repository. The BatchJob includes runtime information collected by a profiling scheduler, more details in section \ref{sec:exampleSchedulingAlgorithm}.

~~~~~~~{#lst:batchJobManifest .yaml caption="Example: Spark BatchJob manifest (shortened)"}
apiVersion: batchjob.gcr.io/v1alpha1
kind: BatchJob
metadata:
  name: spark-crawler
spec:
  externalScheduler:
    profiler:
    - flink-wordcount: 24 (16)
      spark-pi: 27 (16)
      batchjob-sample3: 31 (8)
      #... more information from the external scheduler
  sparkSpec:
    type: Scala
    image: spark-webcrawler:latest
    imagePullPolicy: Always
    mainClass: com.example.webcrawler.WebCrawlerApplication
    mainApplicationFile: "local:///opt/spark/webcrawler.jar"
    arguments:
    - Web_crawler
    - "10000"
    - Kubernetes
    #... more spark specific configuration
~~~~~~~

The concept of a Testbed allows the reservation of resources inside a cluster. It may be unreasonable to give an external scheduler all available resources in a multi-tenant cluster, thus resources available to the external scheduler can be precisely controlled using Testbeds. Listing \ref{lst:testbedManifest} shows an example configuration for a Testbed resource. The Testbed does not directly specify the final size of the Testbed but rather specifies which Nodes to include and how many slots per node should be created. Finally, it also specifies the size of each slot in terms of a CPU and memory request. The size (both in terms of the number of slots and resources) of Testbeds can only be controlled via the Testbeds manifest. Managing the Testbed is not exposed through the Interface, as they are not expected to be changed by an external scheduler, other than using its slots.

~~~~~~~{#lst:testbedManifest .yaml caption="Example: Testbed manifest"}
apiVersion: batchjob.gcr.io/v1alpha1
kind: Testbed
metadata:
  name: profiler-testbed
  namespace: default
spec:
  slotsPerNode: 2
  nodeLabel: "tuberlin.de/node-with-slots-profiler"
  resourcesPerSlot:
    cpu: "700m"
    memory: "896Mi"
~~~~~~~


![Queue with more jobs than available slots. Once Job2 completes Job4](graphics/QueueBased.png){#fig:queueBasedImage short-caption="Example execution of a queue-based Scheduling"}

~~~~~~~{#lst:schedulingManifest .yaml caption="Example: Queue-based Scheduling manifest"}
apiVersion: batchjob.gcr.io/v1alpha1
kind: Scheduling
metadata:
  name: profiler-scheduling-0
  namespace: default
spec:
  queueBased:
    - name: spark-crawler #0
      namespace: default
    - name: batchjob-spark #1
      namespace: default
    - name: batchjob-spark #2
      namespace: default
    - name: spark-crawler #3
      namespace: default
  testbed: # Testbed the scheduling is targeting
    name: profiler-testbed
    namespace: default
~~~~~~~

The External-Scheduler-Interface allows an external scheduler to control which cluster resources batch applications like Apache Spark and Apache Flink will use for their TaskManager and Executor Pods. No assumptions are made about the Driver and JobManager Pods. Schedulings created by the external scheduler are always directed towards a Testbed. Currently, an active scheduling will claim its Testbed and prevent other Schedulings from using it. The Scheduling resource creates a queue of BatchJobs that will be run in the specified Testbed. Schedulings currently support a slot-based strategy and a queue-based strategy. The slot-based strategy specifies which job should use which slots, whereas the queue-based strategy only describes a queue of jobs and no specific slots. Listing \ref{lst:schedulingManifest} shows an example resource manifest for a Scheduling with a queue of four jobs. Once created (assuming all jobs exist), the Interface will create the jobs and instruct them to use the first four free slots of the `profiling-testbed` Testbed. Multiple occurrences of the same BatchJob imply creating the job with multiple executors. Since the queue-based strategy does not reference slots directly, an ordering is used to associate an executor with a slot. The queue size is unlimited; jobs that cannot be executed will be queued. Once slots become available again, they will be associated with jobs in the queue. Figure \ref{fig:queueBasedImage} demonstrates how a queue that exceeds the number of slots would schedule each executor instance. A job is only submitted if all its executors are submitted because the current implementation can not change the number of executors per job during runtime.

The Interface visible to an external scheduler is supposed to be simple. Naturally, an external scheduler could interact with the Interface through the Kubernetes API by managing the scheduling manifest. However, a thin HTTP layer is provided if the external scheduler cannot directly access the Kubernetes API. The External-Scheduler-Interface offers endpoints for querying the current cluster situation in the form of the Testbeds slots occupation status and the status of Schedulings and BatchJobs. The Interface provides a REST API that allows the creation and deletion of Schedulings and the ability to update information stored inside the BatchJob manifests. An additional Stomp [@Stomp] (WebSocket) server is available to not enforce any polling for updates. More concrete information, like Node metrics, can be queried from a metric provider commonly deployed along with the cluster.

The functionality is demonstrated as part of the evaluation, either through the `Manual Scheduler`, which acts as a testing tool and as visualization, and the example scheduler, which uses multiple Testbeds to profile BatchJobs for its scheduling decisions.