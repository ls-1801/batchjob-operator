Volcano is a System Batch-Job Scheduler made for High-Performance Workloads on Kubernetes. Volcano extends Kubernetes with functionalities that Kubernetes do not natively support. Some of these functionalities are critical when working with High-Performance Workloads, like “PodGroups”.
In a scenario where a Framework might want to create multiple Pods for its computation, the resources inside the cluster only allow for a few of them to be deployed. Applications could encounter deadlocks, requiring more Pods to be deployed to progress.
The Concept of PodGroups prevents Pods from being scheduled unless all of them can be scheduled.

The Volcano scheduler is based on the Kubernetes Scheduling Framework, influencing the scheduling cycle at the extension points. 
This way, Volcano can implement many scheduling policies, which High-Performance Batch Applications commonly use.
Volcano is more concerned with policies around the actual scheduling algorithm. In contrast, the External-Scheduling-Interface, introduced in this work, focuses on aiding the development of the actual scheduling algorithm.
In theory, algorithms could be implemented using Volcano, but since Volcano provides just a thin layer above the Kubernetes Scheduling Framework, using Volcano for the development of new scheduling algorithms does not seem like a plausible choice.

With both frameworks supporting Batch Scheduling in different ways, the External-Scheduling-Interface and Volcano could complement each other, but there has been no further investigation.