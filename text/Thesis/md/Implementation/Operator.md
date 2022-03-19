## Batch Job Operator

The Batch Job Operator comprises the Batch Job Reconciler and the Batch Job CRD. The Reconciler is listing for changes regarding the Batch Jobs CRs and Applications CRs, which are managed by the Spark Operator and the Flink Operator.

The Batch Job Operator knows how to construct the corresponding application given the Batch Job CRs specification. With the Spark and Flink Operator, reusing existing software allows the Batch Job CR to be only a thin wrapper around either a Spark or a Flink specification. In addition to the Spark and Flink CR, it may contain additional information that previous invocations of the External-Scheduler have stored.

Currently, the Batch Job CR only contains a partial application-specific specification. The  Batch Job CR requires a user to specify only the Required Components, like Application Image containing the actual application and its arguments like a dataset or where to find it (e.g., using HDFS). Although it would not matter, if a user would submit a fully specified Flink or Spark Application, the Operator would overwrite most of the Driver/Executor Pod specific configuration and replication configuration.

![StateMachine](graphics/batch_job_state_machine.pdf)

The Batch Job Reconciler is implemented as a nested state machine with anonymous sub-states. The Approach was chosen as it creates more comprehensible software, which can be split easier into Components and handle edge cases by design. 

Initially, Batch Jobs submitted to the cluster remain in the Ready State. While in the Ready State, the Batch Job Reconciler will not do anything. A *Scheduling* can acquire a Batch Job. The Batch Job CR will move into the InQueueState until the *Scheduling* instructs the Batch Job Operator to create the application and track its lifecycle or releases it.

Communication between the Batch Job Reconciler and the Scheduling Reconciler is done via the *Batch Jobs* spec (*.spec.activeScheduling* and *.spec.creationRequest*). If *scheduling* wants to claim a Job, it updates the active scheduling spec. This mechanism ensures that only one *Scheduling* at a time can use the *Batch Job*. On the flip side, a *Scheduling* can claim multiple *Batch Jobs*. Suppose the active *scheduling* releases the Job; by removing the activeScheduling spec, the *Batch Job* moves back into the Ready State. Releasing a Job could happen at any time and may even cause any created application to be removed.

Once a Batch Job is in the InQueue State, the Reconciler waits for the creation request issued by the *Scheduling* Reconciler. The request is again done using the *Batch Jobs* spec and specifies desired replication and the *TestBed* and slots the application should use.

When configuring the application to be created by the corresponding Operator, there are two types of configuration. Configuration can either be:

- Persisted inside the Batch Job CR, which is used on every invocation of the application. This includes the Applications Image and arguments, like the data set

- Scheduling dependent. These configurations can not be stored inside the CR and must be supplied with the creation request.

After a Batch Job was requested to create the application, application-specific logic is executed. In any case, the actual steps for deploying the applications to the cluster are done by the Applications Operator (Flink Operator[@FlinkOperator] or Spark Operator[@SparkOperator]). The Batch Job Reconciler only instructs the Application Operators with configurations for the Executor/TaskManager Pods, so they are identifiable to the Extender.

When creating the application, the following aspects are configured for the Executor/TaskManager Pods:

- **Resource Requests**: The Container resources are specified by the Testbeds slot size. For the pods to fit inside a slot, they need the correct Resource Request. (Currently only CPU and Memory)

- **Slot IDs**: The Scheduling (or the external scheduler) decides which slots are used by which Job. For the Executor/TaskManager Pods to be placed into the correct slot (technically the correct node), Pods need to be made identifiable by the Scheduler Extender.

- **replication**: The Number of Executor/TaskManager Pods depends on the Number of Slots that will be used for the application.

- **Priority Class**: Application pods need a *PriorityClass* otherwise, preemption will not be triggered by the Kubernetes Scheduler.

- **Scheduler Name**: Application pods need a *SchedulerName* otherwise, the default KubeScheduler will handle the Scheduling and thus ignore the Scheduling Extender.

Configuration of **Resource Requests**, **Replication** is straightforward, as both the Spark and Flink Operator expose these via their respective CRDs. The Spark Operator actually exposes the complete PodSpec for both driver and executor pods, whereas the Flink Operator only exposes a few PodSpec attributes. The Flink Operator had to be extended with the missing configurations. This way **Resource Requests**, **Replication**, **Priority Class**, **Scheduler Name** are configured.

The difference between any of the mentioned above configurations and the **Slot IDs** is that the Application Operators only allow (rightfully so) to specify a single pod spec. This is because the Executor/TaskManager Pods are controlled by a Stateful Set, which scales up to the desired replication. However, the configurations mentioned above are valid for all pods, but *Slot IDs* need to be different.

This issue can be circumvented by leaving the final decision of which pod goes into which slot to the Extender—submitting only a list of all SlotIDs to the Extender. The Extender needs to decide which pod goes into which slot. Pods are configured with an affinity of the combined set of nodes where the slots reside on.

![Affinities](graphics/affinity.png){height=25%}

Once the application was created, the Job moves into the Submission State. It resides there until all Pods where scheduled, at which point it moves on into the Running State. The underlying Applications state is monitored until it moves into the application-specific completed state (Spark: *Completed* and Flink: *Stopped*). During the implementation, scenarios were encountered in which the Batch Job reconciler was not running. Once restarted, found Applications in a completed state without passing the Scheduling, submission, or running state. To prevent any tight coupling, none of these transitions are required to be considered a successful execution.

The Batch Job reconciler tracks the time an application ran by creating timestamps once it started running and its completion.

## TestBed Operator

The *TestBed* Operator comprises the Reconciler Loop and the *TestBeds* CRD. The *TestBeds* CR is supposed to model a Collection of slots located in a Cluster of Machines. Slots can have specified Resources. While no application is running inside a slot, it is considered *free*. To reserve resources in the cluster and thus guarantee applications supposed to be deployed inside a *free* slot actually to get the resources, the *TestBed* Operator needs to:

- **Reserve Resources** by using so-called Ghost Pods inside the cluster that specify a resource request and thus reserve the resources

- **Preempt** Ghost Pods for pods that wants to be deployed inside a slot

The TestBed Reconciler listens to changes to the TestBed CR and the current cluster situation. It ensures that the correct number of pods with the specified resource requests are always deployed onto the cluster. The TestBed CR is composed of the following configurations:

- Label Name to Identify any nodes that are part of the TestBed. Only the Label Name is specified, not a specific value. The value is later used to create a distinct order of slots in the cluster.

- Number of slots per node

- Resource Request per slot

Given the Testbeds specification, the Reconciler listens for all changes to nodes **with** the specified label. It also needs to listen for nodes **without** any label if the label was removed and the Testbed needs to be resized. Further, it also listens to changes to any pod part of the Test Bed.

The typical Reconciliation Loop works as follows:

- Fetch the current cluster situation

- Calculate the desired cluster situation

- Find the difference. Either delete undesired Pods or create desired Pods


Fetch the current cluster situation by fetching all pods with the **SLOT** label. Pods are then grouped by their node, thus creating a list of pods per node. The desired state is calculated by modeling pods for every slot and grouping them by nodes. When comparing pods, we consider them equal if they reside on the same node, have the same resource request, and have the same *SlotPositionOnNode*.

**Note**: The position of slots on a Node does not matter because slots on are node are only a logical abstraction.

![](graphics/TestBed-ObservedAndDesired.pdf){width=60%}
![](graphics/TestBed-ObservedAndDesired(2).pdf){width=40%}
\begin{figure}[!h]
\begin{subfigure}[t]{0.6\textwidth}
\caption{New Pods need to be Created\label{SlotsNewPods}}
\end{subfigure}
\hfill
\begin{subfigure}[t]{0.4\textwidth}
\caption{Node Change: Pods need to be Deleted\label{SlotsNodeChange}}
\end{subfigure}
\end{figure}


![](graphics/TestBed-ObservedAndDesired(1).pdf){width=50%}
![](graphics/TestBed-ObservedAndDesired(3).pdf){width=30%}
\begin{figure}[!h]
\begin{subfigure}[t]{0.6\textwidth}
\addtocounter{subfigure}{2} % 
\caption{ResourcePerSlot Change: New Pods need to be Created\label{SlotsResourceChange}}
\end{subfigure}
\hfill
\begin{subfigure}[t]{0.4\textwidth}
\caption{Desired State: No Change\label{SlotsDesiredState}}
\end{subfigure}
\caption{TestBed Observed and Desired}
\end{figure}

The Reconciler now builds a set of observed pods and a set of desired pods. \ref{SlotsNewPods} shows an example scenario where the control-loop realizes that pods from the desired state are not in the current state, thus creating the missing pods in the *desired and not existing* set. In a different scenario displayed by \ref{SlotsNodeChange} the label on a node was removed, thus reducing the number of slots inside the Testbed. Pods that are in the *existing and not desired* set will be removed. The final set is the *desired and existing* set, which contains pods that already have the correct resources requirement and are placed on the correct node.

Currently, the SlotOccupationStatus holds the following information: 

- **NodeID** and **NodeName**: which is derived from the Test-Bed Selector Label on the node
- **Position**: which is the SlotID, 
- **slotPositionOnNode**: where the position does unique among the whole Test Bed, SlotPosition on node is only unique per node
- **PodName** and **PodUID**: The Name and the Unique Identifier of a Pod that is currently residing inside the slot
- **state**: is the current state of the slot, which can either be *free*, *reserved*, or *occupied*


## Extender

![Components under control of the External-Interface System\label{ComponentsInControl}](graphics/extender_function.pdf)

Extender Component is integrated within the TestBed Reconciler. Suppose the reconciliation loop detects that the cluster is in progress. The loop is aborted to prevent changes from the Testbed Reconciler and the Extender to act concurrently on the TestBed CR. 
Currently, a cluster is considered in progress if any of the pods require Scheduling (.spec.nodeName is not set) or are terminating (deletion timestamp is set). 

The Extender is the component that directly interacts with the Kubernetes Scheduler. An additional scheduler, with an additional scheduling profile, is running concurrently to the default Kube-Scheduler. The custom scheduler (which will be referred to as Kube-Scheduler) is configured to use the Extender. To guarantee the Scheduling of pods onto the TestBeds slots, the Extender extends the Filter and Preemption extension points of the Kubernetes scheduling cycle. The main problem the Extender can solve, is that the BatchJob Operator does not have full control of pods created downstream by the Applications Operator. 

\ref{ComponentsInControl} shows which of the Components and Resource Managed by them are under the control of the External-Interfaces System. The BatchJob Operator can only control the Application CR created. The Application CR only describes a single PodSpec, which will be later be replicated into multiple pods by the Replication Controller, which is part of the StatefulSet. Thus it is not possible to set Pod specific configurations, like the SlotID, at the BatchJob Operator Level. However all Pods can be configured, with enough information, for the Extender to figure out which pod belongs in which slot.

**Note**: PodSpec here only refers to the TaskManager/Executor PodSpec, as the External-Interface does not handle Scheduling of the JobManager/Driver Pods.

In order to influence the scheduler to schedule pods onto nodes with the correct slot, the number of possible nodes is first limited by all nodes containing any of the slots using affinities.

The Applications PodSpec created by the BatchJob Operator, limits the possible nodes, the kube-scheduler can use affinities. During development multiple scenarios, of interaction between kube-scheduler and Extender were identified. 

![Kube-Scheduler limits nodes, Extender selects node with slot\label{ExtenderFiltering}](graphics/ExtenderFiltering.pdf){height=50%}

If the kube-scheduler, detects not enough available resources, it will trigger preemption. If no preemption is required the Filter endpoint of the Extender is queried, to further limit the possible nodes. \ref{ExtenderFiltering} describes how the set of possible nodes is first shrunk by affinities and later the correct node is chosen by the Extender, based on the first free slot. Once a pod invokes the Extender a slot will be reserved. The Extender Stores Information inside the TestBed CRs Status. To prevent race conditions between the TestBed Reconciler and the Extender mutual access to the TestBed CR, is required, which is guaranteed since the TestBed Reconciler will abort its reconciliation if Scheduling is in progress.

In the Event no resources are available the kube-scheduler invokes the preemption endpoint of the Extender. Preemption is straightforward, since the TestBed already reserves resources using ghost pods. The Extender chooses a slot based on the first free slot, and returns the currently residing ghost pod to be the victim.

For the TestBed Reconciler to detect scheduled pods the Extender will in addition to returning the target node to the scheduler, also sets the **SlotID** + **NonGhostPod** of scheduled pods. Once Scheduling has finished, the TestBed Reconciler will no longer abort execution, and preempt all ghost pods, that were not previously preempted by the Kube-Scheduler. Further Information about the Extender can be found inside the source code, as it contains heavily commented code, to describe different scenarios.

## Scheduling Operator
Once again the Scheduling Operator is composed of the Reconciler Loop and the Scheduling CRD. The Scheduling Reconciler like the BatchJob Reconciler implemented using a nested Statemachine. 
The Scheduling CR models, a collection of jobs, and a slot selection strategy. Once submitted the Reconciler first acquires all jobs, and starts running them. The Scheduling tracks the execution of all its jobs and submits new Jobs once old jobs have finished and slots become available again. Initially the Scheduling was planned to only support offline Scheduling, where an external scheduler plans the execution of multiple jobs in advance, however in theory updating the Scheduling spec would allow an online scheduling, but in the current state it is rather unreliable, as it only allows jobs to be added to the end of the queue.

Slot selection strategy do not aim to provide a full scheduling algorithm, they are just means for an external scheduler to describe which Job should use which slot. 


- Implemented as a state machine
- Acquire State claims all Batch Jobs and the Test Bed
- Once all Jobs are in the InQueueState scheduling choses the first n runnable jobs
- Two Modes: SlotBased + QueueBased (Images)
- Once Creation was Requested, Reconciler waits until all jobs submitted were scheduled.
- This is required, because the Slot Reservation is not instantaneous. Wait for Batch Job Reconciler + Application Operator until the Extender marked slots as reserved.
- At this point the Scheduling waits until slots come available, different Modes require different Condition
 - The Queue Based Scheduling, only requires a number of available slots
 - Slot Base scheduling requires specific slots to come available

- Once the Queue is empty the Scheduling moves into the await completion state until all jobs have completed

- Note: Online Scheduling: is possible by updating the scheduling CR and extending the Queue


## External Scheduler Interface

- Interaction with the Scheduling Interface is naturally done via the Kubernetes API, creating, updating, deleting CRs. 
- If the external-scheduler chooses not to directly interact with kubernetes, a thin layer in form of web api is provided.
- The interface aims to abstract away some of the Kubernetes features like namespaces. 
- The interface allows to create update and delete schedulings. Query for jobs inside the cluster. Query for slots inside the cluster.
- The interface contains a web socket server that broadcasts changes to jobs, schedulings, Testbed


