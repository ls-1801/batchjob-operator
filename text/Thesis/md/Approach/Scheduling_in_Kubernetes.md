In this section, the Scheduling Model of Kubernetes will be introduced as it is a vital part of the implementation for the Interface. The scheduling Problem in Kubernetes is the problem of deciding which Pods are running on which machine. Kubernetes runs workloads by placing containers into Pods to run on Nodes. A node may be a virtual or physical machine, depending on the cluster. Each node is managed by the control plane and contains the necessary software to run Pods [@Configuring_The_Built-In_2022_kubernetes]. For some Pods, the question can be easily solved. For example, Pods controlled by a DaemonSet [@Configuring_The_Built-In_2022_kubernetes] are by the specification of the DaemonSet running on every Node. Without further information, a feasible choice of scheduling Pods onto Nodes seems to be simple round-robin scheduling. Every Pod that requires scheduling gets scheduled onto the next Nodes until all Nodes have Pods then the cycle is repeated. However, both Pods and Nodes can influence the scheduling. 

Pods can specify the resources they are going to use and may even set a Hard Requirement in the form of a request for resources they require to run. Scheduling needs to take the resources requests into account when scheduling a Pod across Nodes. Pods can also directly influence the Node they should be scheduled on, either through specifying a *NodeName* directly, a *NodeSelector*, which identifies possible multiple Nodes, or a more general concept of *affinity*. 
Affinities provide the ability to set hard and soft requirements, where a Pod may become unschedulable if a hard requirement cannot be met, and a soft requirement is not preferred but an acceptable decision. With Affinities, even inter-pod Affinities can be specified and Pods can choose not to be scheduled on a Node if another Pod is already deployed. 

On the other side, Nodes can also specify so-called Taints, where only Pods with the fitting Toleration can be scheduled onto the Node. Kubernetes also use the Taint mechanism to taint Nodes that are affected by Network problems, Memory-Pressure, or are not ready after a restart. The Taint specifies if just new Pods cannot be scheduled *NoSchedule* onto the Node, or if even already running Pods should be evicted *NoExecute*. Tainted Nodes with either type of Taints are not chosen during scheduling.

However, Pods can bypass the Taint Node if they have the fitting Toleration. For example, if a Node becomes unreachable due to a network outage, Nodes affected will be tainted with the *node.kubernetes.io/unreachable"* Taint, which is set to *NoExecute*. Pods can neither be scheduled onto the unreachable Nodes nor keep running without the fitting Toleration. In a likely scenario where the network recovers and the machine becomes reachable again, it would be inefficient to restart all Pods instantly if a machine becomes unreachable for a short time. To prevent Pods from being evicted (and thus most likely restarted), Kubernetes by default creates the unreachable Toleration for all Pods. However, the Toleration set by Kubernetes also specifies a maximum duration where the taint is tolerated (by default, 5 minutes). If a Node stays unreachable for more than 5 minutes, all Pods are evicted.

In the context of scheduling, Taints, Tolerations, Pods, and Nodes only refer to the objects stored inside the distributed key-value store. If a physical machine becomes unreachable, the objects manifest is updated. Whether the physical containers are still running or not cannot be determined, but for the duration of the Toleration, they do count as running and are not replaced with new Pods.

## Scheduling Cycle
Kubernetes Scheduling is based on the Kubernetes Scheduling Framework. As the extensible nature of Kubernetes, the Scheduling framework is also extensible. Customization to the scheduling can be done at so-called Extension Points. 

![Scheduling Cycle \label{PodSchedulingContext}](graphics/scheduling-framework-extensions.png){width=25%, height=25%}

Each attempt of scheduling Pods to Nodes is split into two phases, the *Scheduling Cycle*, and the *Binding Cycle*.

The Scheduling Cycle is more interesting in the context of this work. In this part of the cycle, the decision is made on which Node to schedule which Pod. The Binding Cycle is the scheduling phase, where the Pod is being deployed to the physical machine. Multiple Scheduling Cycles will not run concurrently, as this would require synchronization between multiple Scheduling Cycles. Since the Scheduling Cycle is relatively fast, executing multiple Scheduling Cycles in parallel seems unnecessary. However, the Binding Cycle requiring the deployment of Pods is compared to the Scheduling Cycle rather long-living and thus may be executed in parallel. Both the Scheduling Cycle and the Binding Cycle may abort the scheduling of a Pod in case it may be unschedulable.

The Pod Scheduling Context can be modified through plugins that intern use the Scheduling Framework's extension points. The graphic \ref{PodSchedulingContext} shows the available extension points, which are the Pod Scheduling Cycle phases. Multiple Plugins can be registered for any of the different extension points and will run in order. 

This section will give a brief breakdown of the different stages and highlight those critical in the implementation of the External-Scheduler-Interface.

**Queue Sort**: Sometimes, multiple Pods require a scheduling decision. The Queue Sort Extension point extends the Scheduling Cycle with a comparison function that allows the Queue of Pods to wait for a scheduling decision to be sorted. Usually, multiple Plugins can influence the scheduling decision at a time; however, having numerous Plugins sorting the queue of waiting Pods will not result in anything meaningful. Thus, only a single plugin can control the queue at a time. After sorting the Queue of Pods, the first Pod is chosen and passed onto the next stage.


**Filter**: The Scheduling decision is made by first using the PreFilter and Filter extension point. The Filter Extension point filters out Nodes that cannot run the Pod. Most extension points also have a pre-extension point used to prepare information about the Pod. As mentioned earlier, any extension point can return an error indicating that the Pod may be unschedulable.

The Filter extension point also has a **Post-Filter** Extension, which is only called if no Node passes the Filter, and a Pod becomes *unschedulable*. The Post-Filter Extension can then be used to find a possible Scheduling using preemption. Preemption is very important, as it is required to model Pods having a higher priority than other Pods. In this work, preemption is required to guarantee that the scheduling from an External-Scheduler is correctly applied even if other Pods in the cluster exist that are unknown to the External-Scheduler.

**Scoring**: The filter plugin is concerned with hard requirements that prevent Pods from being scheduled. If previous filter plugins deem multiple Nodes suited for the Pod to be scheduled on, a decision needs to be made to find the best Node. The scoring plugin considers soft requirements and gives Nodes a lower or higher score that can or cannot fulfill the soft requirements. Finally, all Scores are normalized between two fixed values.

**Reserve**: The reserve phase is used to reserve resources on a Node for a given Pod. This is required due to the asynchronous transition into the Binding Cycle. Pods that are supposed to be bound to a Node invoke reservation to prevent possible race conditions between future Scheduling Cycles. Reservation may fail, in which case the cycle moves to the unreserve phase, and all previously completed reservations are revoked. Usually, the Reservation extension point is used for applications that will not use the default containerization mechanism of Kubernetes and rely on different binding mechanisms.

**Permit**: The final stage of the Scheduling Cycle is the Permit Stage, which ultimately denies or delays the binding of a Pod in a case where binding might not be possible or still requires time.

## Extending the Scheduler

Common ways of extending the Scheduler are either to implement a custom Scheduler and to replace the KubeScheduler in the cluster,
or to use the *Extender* API that instructs the Scheduler to invoke an external API for its scheduling decision.

The pluggable architecture of the Scheduler allows plugins to extend the scheduling cycle at the extension points. Different Scheduler profiles can use different plugins. Profiles can be created or modified using a *KubeSchedulerConfiguration*. Only Pods that specify the Scheduler Profile using *.spec.schedulerName* are scheduled using the profiles.

The Extender Mechanism describes an external Application waiting for HTTP Requests issued during the Scheduling Cycle to influence the scheduling decision. Instructing the Scheduler to use an Extender, in the current Implementation of Kubernetes, is not limited to a specific Profile, but all Profile will use the configured Extender. An Extender can specify the Verbs it supports. Verbs in this context refer to Filtering, Scoring, Preempting, and Binding.

Note: This work does not replace the KubeScheduler, but deploys a Second Scheduler to the cluster with a Profile that uses the Extender API. This is done because of the limitation that every scheduling profile will use the Extender, which seems unnecessary in the context of a prototype.

