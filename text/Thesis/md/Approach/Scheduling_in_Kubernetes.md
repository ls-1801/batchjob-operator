In this section, the Scheduling Model of Kubernetes will be introduced as it is a vital part of the implementation for the Interface.

The smallest unit of *deployment* in Kubernetes is a Pod[@kubernetesPodDocs]. While a pod may consist have multiple containers. Containers in a pod are guaranteed to run on the same node.
Containers in Pods also share storage and network resources across the container boundary. In General, containers inside the same Pod are tightly coupled and commonly used in a sidecar pattern to extend the main container with common functionalities across the cluster, like the Kube-RBAC-Proxy[@rbacProxy] that is frequently used with Containers that interact with the Kubernetes API and require authorization.

Usually, in Kubernetes, Pods are not created by themself but are managed by resources that build on top of them. Most commonly, Pods are used in Combination with Jobs, Deployments, or Statefulsets, which control the Lifecycle of the Pod.

Resources like Jobs, Deployments, and Statefulsets describe Pods' desired state. Essentially Kubernetes has controllers that monitor both changes to the Resources Manifest and the current cluster situation. 
The Control-Loop allows a Deployment consisting of a Pod template with a Replication-Factor to the exact amounts of pods running inside the cluster. If any of the Pods fails, the Deployment Controller creates a new one. But on the Flip-Side, the controller also knows when the Resources Manifest is updated. For example, if the container is updated to a more recent version, all containers need to be replaced with the newer version. Deployment Resources have many policies that dictate how these actions should happen. Usually, restarting all Pods at once would not be desired so that the deployment would allow for Rolling-Upgrades. The controller ensures that new Pods are created first and become ready before old Pods are deleted.

The scheduling Problem in Kubernetes is the problem of deciding which pods are running on which node. For some pods, the question can be easily solved. For example, Pods that a DaemonSet controls are by the specification of the DaemonSet running on every node. Without further information, a feasible choice of scheduling pods onto nodes seems to be simple round-robin scheduling. Every Pod that requires scheduling gets scheduled onto the next nodes until all nodes have pods then the cycle is repeated. However, both pods and nodes can influence the scheduling. 


Pods can specify the resources they are going to use and may even set a Hard Requirement in the form of a request for resources they require to run. Scheduling needs to take the resources requests into account when scheduling a pod across nodes. Pods can also directly influence the node they should be scheduled on, either through specifying a *nodeName* directly, a *nodeSelector*, which identifies possible multiple nodes, or a more general concept of *affinity*. 
Affinities provide the ability to set Hard and Soft Requirements, where a Pod may become unschedulable if a Hard Requirement cannot be met, and a Soft requirement is not preferred but an acceptable decision. With Affinities, even inter-pod affinities can be specified where pods can choose not to be scheduled on a node where another pod is already deployed. 

On the other side of the scheduling equation, nodes can also specify so-called Taints, where only Pods with the fitting toleration can be scheduled onto the node. Kubernetes also use the Taint mechanism to taint nodes that are affected by Network problems, Memory-Pressure, or are not ready after a restart. The taint specifies if just new pods cannot be scheduled *NoSchedule* onto the node, or if even already running pods should be evicted *NoExecute*. These nodes with these taints should usually not be chosen during scheduling.

However, Pods can bypass the Taint node if they have the fitting toleration. For example, if a node becomes unreachable due to a network outage, nodes affected will be tainted with the *"node.kubernetes.io/unreachable"* taint, which is set to *NoExecute*.
Without the fitting toleration, Pods can not be scheduled onto the unreachable nodes, but they may still keep executing while running on an unreachable node. This is possible because Kubernetes by default creates the unreachable toleration for all pods (unless specified otherwise). However, the toleration set by Kubernetes also specifies a duration where the taint is tolerated (by default, 5 minutes).

Quick Side Note: Taints and Tolerations, and Pods and Nodes, here all refer to the resources stored inside etcd. If a physical node becomes unreachable, the Resource is updated with the Manifest. If the physical containers are still running or cannot be determined, but for the duration of the toleration, they do count as running and are not replaced by new Pods.

## Scheduling Cycle
Kubernetes Scheduling is based on the Kubernetes Scheduling Framework. As the extensible nature of Kubernetes, the Scheduling framework is also extensible. Customization to the scheduling can be done at so-called Extension Points. 

![Scheduling Cycle](graphics/scheduling-framework-extensions.png){width=25%, height=25%}

Each attempt of scheduling pods to nodes is split into two phases, the *Scheduling Cycle*, and the *Binding Cycle*.

The Scheduling Cycle is more interesting in the context of this work. In this part of the cycle, the decision is made on which node to schedule which Pod. The Binding Cycle is the scheduling phase, where the Pod is being deployed to the physical machine. Multiple Scheduling Cycles will not run concurrently, as this would require synchronization between multiple Scheduling Cycles. Since the Scheduling Cycle is relatively fast, executing multiple Scheduling Cycles in parallel seems unnecessary. However, the Binding Cycle requiring the deployment of Pods is in comparison to the Scheduling Cycle rather long-living and thus may be executed in parallel. Both the Scheduling Cycle and the Binding Cycle have in common is that both may abort the scheduling of a pod in case it may be unschedulable.

As mentioned earlier, the scheduling context can be modified through plugins that intern use the Scheduling Framework's extension points. The Pod Scheduling Cycle can be broken into a Filter, Scoring Reservation and Permit State. Multiple Plugins can be registered for any of the different extension points and will run in order.

However, the first Extension point is the Queue Sort Extension point. Sometimes multiple pods require a scheduling decision. The Queue Sort Extension point extends the Scheduling Cycle with a comparison function that allows the Queue of Pods to wait for a scheduling decision to be sorted. Usually, multiple Plugins can influence the scheduling decision at a time; however, having numerous Plugins sorting the queue of waiting pods will not result in anything meaningful. Thus, only a single plugin can control the queue at a time.

The following way to influence the Scheduling decision is done using the PreFilter and Filter extension point. The Filter Extension point filters out Nodes that cannot run the Pod. Most extension points also have a pre-extension point used to prepare information about the Pod. As mentioned earlier, any extension point can return an error indicating that the Pod may be unschedulable.

The Filter extension point also has a PostFilter Extension, which is only called if after the Filtering finds no Node. The PostFilter Extension can then be used to find a possible Scheduling using preemption. Preemption is very important, as it is required to model Pods having a higher priority than other Pods. In the context of this work, preemption is required to guarantee that the scheduling from an external scheduler is correctly applied even if other pods in the cluster exist that are unknown to the external scheduler.

The next available extension point, part of the Scheduling Cycle, is the Scoring point. The filter plugin is concerned with hard requirements that prevent pods from being scheduled. If previous filter plugins deem multiple nodes suited for the Pod to be scheduled on, a decision needs to be made to find the best node. The scoring plugin considers soft requirements and gives nodes a lower or higher score that can or cannot fulfill the soft requirements. Finally, all Scores are normalized between two fixed values.

The reserve phase is used to reserve resources on a node for a given pod. This is required due to the asynchronous transition into the Binding Cycle. Pods that are supposed to be bound to a node invoke reservation to prevent possible race conditions between future Scheduling Cycles. Reservation may fail, in which case the Cycle moves to the unreserve phase, and all previously completed reservations are revoked. Usually, the Reservation extension point is used for applications that will not use the default containerization mechanism of Kubernetes and rely on different binding mechanisms.

The final state of the Scheduling Cycle is the Permit state. The Permit State can ultimately deny or delay the binding of a Pod in a case where binding might not be possible or still requires time.

## Extending the Scheduler

Common ways of extending the Scheduler are either to implement a custom Scheduler and to replace the KubeScheduler in the cluster,
or to use the *Extender* API that instructs the Scheduler to invoke an external API for its scheduling decision.

The pluggable architecture of the Scheduler allows plugins to extend the scheduling cycle at the extension points. Different Scheduler profiles can use different plugins. Profiles can be created or modified using a *KubeSchedulerConfiguration*. Only Pods that specify the Scheduler Profile using *.spec.schedulerName* are scheduled using the profiles.

The Extender Mechanism describes an external Application listing for HTTP Requests issued during the Scheduling Cycle to influence the scheduling decision. Instructing the Scheduler to use an Extender, in the current Implementation of Kubernetes, is not limited to a specific Profile, but all Profile will use the configured Extender. An Extender can specify the Verbs it supports. Verbs in this context refer to Filtering, Scoring, Preempting, and Binding.

*(TODO: Check)*
Note: This work does not replace the KubeScheduler, but deploys a Second Scheduler to the cluster with a Profile that uses the Extender API. This is done because of the limitation that every scheduling profile will use the Extender, which seems unnecessary for this work.

