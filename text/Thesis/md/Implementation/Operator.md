In this section, the Operators controlling CRDs introduced in the previous section are discussed.

## BatchJob Operator

The BatchJob Operators control-loop is listing for changes regarding the BatchJobs CRs and applications CRs managed by the Spark Operator and the Flink Operator.

The BatchJob Operator knows how to construct the corresponding application given the BatchJob CRs specification. The lifecycle of the application CR is delegated the the applications Operator. Reusing existing software allows the BatchJob CR to be only a thin wrapper around either a Spark or a Flink specification. In addition to the Spark and Flink CR, it may contain additional information that previous invocations of the external scheduler have stored.

The  BatchJob CR requires a user to specify only the required components, like the application image containing the actual application and its arguments like a dataset or where to find it (e.g., using HDFS). Currently, the BatchJob CR only includes a partial application-specific specification. It does not matter if a user submits a fully specified Flink or Spark application because the Operator would overwrite most of the Driver/Executor Pod specific configuration and replication configuration.

![BatchJob State machine](graphics/batch_job_state_machine.pdf){#fig:batchJobStatemachine}

 Figure \ref{fig:batchJobStatemachine} shows that the BatchJob reconciler is implemented as a nested state machine with anonymous sub-states. The approach was chosen because it creates more comprehensible software, which can be split into components and handle edge cases by design.

Initially, BatchJobs submitted to the cluster remain in the ready state. While in the ready state, the BatchJob reconciler will not do anything. A Scheduling can acquire a BatchJob. The BatchJob CR will move into the in-queue state until the Scheduling instructs the BatchJob Operator to create the application and track its lifecycle or releases it.

The BatchJob Operator and the Scheduling Operator communicate via the BatchJobs spec (`.spec.activeScheduling` and `.spec.creationRequest`). If a Scheduling wants to claim a job, it updates the active scheduling spec. This mechanism ensures that only one Scheduling at a time can use the BatchJob. On the flip side, a Scheduling can claim multiple BatchJobs. Suppose the active Scheduling releases the job by removing the `activeScheduling` spec, the BatchJob moves back into the ready state. Releasing a job could happen at any time and may even cause any created application to be removed.

Once a BatchJob is in the in-queue state, the Operator waits for the creation request issued by the Scheduling Operator. The request is again done using the BatchJobs spec and specifies desired replication and the Testbed and slots the application should use.

When configuring the application to be created by the corresponding Operator, there are two types of configuration: configurations specific to the Distributed Dataflow application and configurations specific to the current Scheduling. The application configurations are persisted inside the BatchJob CR. These configurations are used on every invocation of the application and include the program and its arguments. As the name implies, Scheduling-dependent configurations need to be changed depending on the Scheduling and thus must be supplied with the creation request.

After a BatchJob was requested to create the application, application-specific logic is executed. In any case, the actual steps for deploying the applications to the cluster are done by the applications Operator (Flink Operator [@FlinkOperator] or Spark Operator [@SparkOperator]). The BatchJob reconciler only instructs the application Operators with configurations for the Executor/TaskManager Pods, so they are identifiable to the Extender.

When creating the application, the following aspects are configured for the Executor/TaskManager Pods:

**Resource Requests**: The Testbeds slot size specifies the container resource request. For the Pods to fit inside a slot, they need the correct resource request (currently only CPU and memory).

**Testbed Name**: The name of the Testbed. Testbeds are not limited to a single Testbed per cluster, so they must be distinguished from another.

**Slot IDs**: The Scheduling (or the external scheduler) decides which slots are used by which job. For the Executor/TaskManager Pods to be placed into the correct slot (technically the correct Node), Pods need to be identifiable by the scheduler extender.

**Replication**: The number of Executor/TaskManager Pods depends on the number of slots that will be used for the application.

 **Priority Class**: The Kubernetes scheduler will not trigger preemption unless the application Pods specifies a `.spec.priorityClassName`.

 **Scheduler Name**: The default Kube-Scheduler is not configured to use the Extender. Application pods need to specify the `spec.schedulerName` of the second Kubernetes scheduler, which is configured with the Extender.

Configuration of **Resource Requests**, **Replication** is straightforward, as both the Spark and Flink Operator expose these via their respective CRDs. The Spark Operator exposes the complete PodSpec for Driver and Executor Pods, whereas the Flink Operator only exposes a few PodSpec properties. The Flink Operator had to be extended with the missing configurations. This way **Resource Requests**, **Replication**, **Priority Class**, **Scheduler Name** are configured.

**SlotIDs** and the **Testbed Name** are not part of the PodSpec, so they are configured using Labels. Labels are a generic list of key/value pairs on every object in Kubernetes.

![Usage of Affinities: Pods can only potentially be scheduled onto Nodes that hosts any desired slot](graphics/affinity.png){#applicationSpecCreation short-caption="Usage of Affinities"}

Ideally, the BatchJob Operator gives every Executor/TaskManager Pod individually a slot ID. The issue is that the application-specific CRs only configure a Pod template. Pods created by the application Operator are managed by a StatefulSet (similar to a ReplicaSet), ensuring the desired replication. This issue is circumvented by leaving the final decision of which Pod goes into which slot to the Extender. The Operator can only configure a list with all **SlotIDs** to the Extender. Figure \ref{applicationSpecCreation} shows how using Affinities limits the set of possible Nodes. Any Node that contains any of the desired slots needs to be considered by the scheduler and the Extender.

Once the application is created, the job moves into the submitted state. It resides there until all Pods where scheduled, at which point it moves on into the running state. The underlying applications state is monitored until it moves into the application-specific completed state (Spark: `Completed` and Flink: `Stopped`). During the implementation, scenarios were encountered in which the BatchJob reconciler was not running. Once restarted, it found applications in a completed state without passing the Scheduling, submission, or running state. To prevent any tight coupling, none of these transitions are required to be considered a successful execution.

The BatchJob reconciler tracks the time an application ran by creating timestamps once it started running and its completion.

## Testbed Operator

The Testbed Operator monitors changes to Pods and Nodes in addition to controlling Testbed CRs. The Testbeds CR is supposed to model a collection of slots located in a cluster of machines. Slots can have specified resources. While no application is running inside a slot, it is considered *free*. To reserve resources in the cluster and thus guarantee applications supposed to be deployed inside a *free* slot actually to get the resources, the Testbed Operator needs to:

- **Reserve Resources** by using so-called `Ghost Pods` inside the cluster that specify a resource request and thus reserve the resources

- **Preempt** Ghost Pods for Pods that wants to be deployed inside a slot

The Testbed reconciler listens to changes to the Testbed CR and the current cluster situation. It ensures that the correct number of Pods with the specified resource requests are always deployed onto the cluster. The Testbed CR is composed of the following configurations:

- Label Name to identify any Nodes that form the Testbed. Only the label name is specified, not a specific value. The value is later used to create a distinct order of slots in the cluster.

- Number of slots per Node

- Resource request per slot

Given the Testbeds specification, the reconciler listens for all changes to Nodes **with** the specified label. It also needs to listen for Nodes **without** any label if the label was removed and the Testbed needs to be resized. Further, it also listens to changes to any Pod part of the Testbed.

The typical reconciliation loop works as follows:

- Fetch the current cluster situation

- Calculate the desired cluster situation

- Find the difference. Either delete undesired Pods or create desired Pods


Fetch the current cluster situation by fetching all Pods with the **SLOT** label. Pods are then grouped by their Node, thus creating a list of Pods per Node. The desired state is calculated by modeling Pods for every slot and grouping them by Nodes. When comparing Pods, we consider them equal if they reside on the same Node, have the same resource request, and have the same *SlotPositionOnNode*.

**Note**: The position of slots on a Node does not matter because slots on are Node are only a logical abstraction.

![](graphics/TestBed-ObservedAndDesired.pdf){width=60%}
![](graphics/TestBed-ObservedAndDesired(2).pdf){width=40%}
\begin{figure}[!h]
\begin{subfigure}[t]{0.6\textwidth}
\caption{New Pods need to be created\label{SlotsNewPods}}
\end{subfigure}
\hfill
\begin{subfigure}[t]{0.4\textwidth}
\caption{Node Change: Pods need to be deleted\label{SlotsNodeChange}}
\end{subfigure}
\end{figure}


![](graphics/TestBed-ObservedAndDesired(1).pdf){width=50%}
![](graphics/TestBed-ObservedAndDesired(3).pdf){width=30%}
\begin{figure}[!h]
\begin{subfigure}[t]{0.6\textwidth}
\addtocounter{subfigure}{2} % 
\caption{ResourcePerSlot Change: New Pods need to be created\label{SlotsResourceChange}}
\end{subfigure}
\hfill
\begin{subfigure}[t]{0.4\textwidth}
\caption{Desired State: No change\label{SlotsDesiredState}}
\end{subfigure}
\caption{Testbed Observed and Desired}
\end{figure}

The reconciler now builds a set of observed Pods and a set of desired Pods. \ref{SlotsNewPods} shows an example scenario where the control loop realizes that Pods from the desired state are not in the current state, thus creating the missing Pods in the *desired and not existing* set. In a different scenario displayed by \ref{SlotsNodeChange} the label on a Node was removed, thus reducing the number of slots inside the Testbed. Pods that are in the *existing and not desired* set will be removed. The final set is the *desired and existing* set, which contains Pods that already have the correct resources requirement and are placed on the correct Node.

Currently, the SlotOccupationStatus holds the following information: 

- **NodeID** and **NodeName**: which is derived from the Testbed Selector Label on the Node
- **Position**: which is the SlotID, 
- **slotPositionOnNode**: where the position does unique among the whole Test Bed, SlotPosition on Node is only unique per Node
- **PodName** and **PodUID**: The Name and the Unique Identifier of a Pod that is currently residing inside the slot
- **state**: is the current state of the slot, which can either be *free*, *reserved*, or *occupied*


## Extender

![Components under control of the External-Interface System\label{ComponentsInControl}](graphics/extender_function.pdf)

Extender Component is integrated within the Testbed Reconciler. If the reconciler detects that the cluster is in progress, it pauses its control loop. This prevents both components from acting on the same Testbed concurrently, thus preventing any unexpected changes to the Testbed slot occupation state while the Extender looks for free slots.
Currently, a cluster is considered in progress if any Pods require Scheduling (.spec.NodeName is not set) or are terminating (deletion timestamp is set).

The Extender is the component that directly interacts with the Kube-Scheduler. An additional scheduler, with an additional scheduling profile, is running concurrently to the default Kube-Scheduler. The custom scheduler (referred to as Kube-Scheduler) is configured to use the Extender. To guarantee the Scheduling of Pods onto the Testbeds slots, the Extender extends the Filter and Preemption extension points of the Kubernetes scheduling cycle. The main problem the Extender can solve is that the BatchJob Operator does not have full control of Pods created downstream by the applications Operator. 

Figure \ref{ComponentsInControl} shows which of the components and resources managed by them are under the control of the External-Interfaces System. The BatchJob Operator can only control the Application CR created. The Application CR only describes a single PodSpec, which will later be replicated into multiple Pods by the replication controller, part of the StatefulSet. Thus, it is impossible to set Pod-specific configurations, like the SlotID, at the BatchJob Operator Level. However, with enough information, all Pods can be configured for the Extender to figure out which Pod belongs in which slot.

**Note**: PodSpec here only refers to the TaskManager/Executor PodSpec, as the External-Interface does not handle Scheduling of the JobManager/Driver Pods.

![Kube-Scheduler limits Nodes, Extender selects Node with slot](graphics/ExtenderFiltering.pdf){#ExtenderFiltering short-caption="Interaction between Extender and Kube-Scheduler"}

Figure \ref{ExtenderFiltering} shows how the Kube-Scheduler is influenced to schedule Pods onto Nodes with the correct slot. The number of possible Nodes is first limited by all Nodes containing any of the slots using affinities. Finally, the Extender chooses the right Node with the designated slot. The first step is already implemented inside the Kube-Scheduler. The second step is elaborated in more detail.
Figure \ref{ExtenderFiltering} shows how the Kube-Scheduler is influenced to schedule Pods onto Nodes with the correct slot. The number of possible Nodes is first limited by all Nodes containing any of the slots using affinities. Finally, the Extender chooses the right Node with the designated slot. The first step is already implemented inside the Kube-Scheduler. The second step is elaborated in more detail.

During development, multiple scenarios of interaction between Kube-Scheduler and Extender were identified. 

If the Kube-Scheduler detects that neither of the possible Nodes has enough resources available, it will immediately trigger preemption, thus forgoing the Filter-Extender. 
If no preemption is required, the Filter-Extender is queried to limit the possible nodes further. The Filter-Extender arguments only contain Nodes that match the resource requests requirement of the Pod. If the Kube-Scheduler offers Nodes that the Extender already used for other Pods, the Filter-Extender limits the possible Nodes to an empty set, triggering preemption.

The Kube-Scheduler has its internal default preemption algorithm. It simulates scenarios of preempting pods with a lower priority until any Node passes its filters (usually the resource request filter). If the internal preemption does not find a possible scenario, preemption is canceled, and the Pod becomes unscheduable. This scenario skips the Preemption-Extender entirely and creates a problem for the External-Scheduler-Interface system. Pods created downstream by the BatchJob Operator are configured with a higher priority than Ghost Pods to prevent the scenario from occurring, thus guaranteeing that preemption will always query the Preemption-Extender.

The Preemption-Extenders arguments include the preemptor Pod and possible preemptees. Because internally, the Kube-Scheduler stops when finding the first potential victim for preemption, the preemptees are only a suggestion and are ignored by the Preemption-Extender. 

The Preemption-Extender uses the **Slot-IDs** and the **Testbed Name** that the BatchJob Operator placed on the Pod. It first searches through all desired slots of the Testbed to find slots that have already been reserved for the Pod. If no Slot has been reserved yet, the Extender finds the next possible free slot. A scenario where no slot is found should not happen because the controlling Scheduling will only submit new jobs once enough slots become available.

Since the Kube-Scheduler could invoke the Extender multiple times for the same Pod, the first invocation reserves a Slot, and subsequent requests will always return the same slot. Slots are marked as reserved using the Testbed CR. Because the Testbed is based around preempting Ghost Pods that reserve system resources, both the Filter-Extender and the Preemption-Extender will prepare preemption internally. At the end of either scenario, the Pod that requires scheduling will have its **Slot-ID** set, and the Ghost Pod previously residing inside the slot is either marked as preempted or removed. The Pod will also be marked as **NonGhostPod**, so the Testbed Operator can preempt the Ghost Pod with the same Slot-ID if it was not deleted through the Kube-Schedulers preemption.

## Scheduling Operator

![Scheduling state machine](graphics/scheduling_state_machine.pdf){#fig:schedulingStateMachine}

Like the BatchJob reconciler, the Scheduling Reconciler is implemented using a nested state machine (Figure \ref{fig:schedulingStateMachine}). In addition to changes to Scheduling resources, the reconciliation loop is also triggered on changes to BatchJobs or Testbeds.
 
The Scheduling CR contains a collection of jobs, a slot selection strategy, the target Testbed. The Scheduling tracks the execution of all its jobs and submits new Jobs once old jobs have finished and slots become available again. 

Initially, the Scheduling was planned only to support offline Scheduling, where an external scheduler plans the execution of multiple jobs in advance. However, in theory, updating the Scheduling spec would allow online scheduling. Still, it is rather unreliable in the current state, as it only allows jobs to be added to the end of the queue.

Slot selection strategies do not aim to provide a full scheduling algorithm. They are just means for an external scheduler to describe which Job should use which slot. 

The lifecycle of a scheduling is described using a state machine.
Once the Scheduling CR is created, it moves into the Acquire state. 
While in the Acquire state, the scheduling tries to acquire all its jobs and its Testbed. If it detects that any of them is currently in use, it will release all of them to prevent any deadlocks. It remains in the acquire state until all acquired BatchJobs move into the InQueue State.

The core submission loop of the Scheduling reconciler moves from the Confirmation state to the Submission State until the queue becomes empty.

Two distinct states are needed between submissions of jobs because the Testbeds slot occupation status is not updated immediately. The Confirmation state awaits previously submitted jobs from confirming their scheduling (BatchJob in Scheduled state). Once previous submissions are no longer pending, the Scheduling waits for slots to become available.

Depending on the Strategy, either specific slots or just a number of slots need to be available to submit the next job from the queue.

Once the queue becomes empty, the core submission loop is exited, and the Scheduling moves into the AwaitCompletion. If all jobs are completed, Scheduling is completed.

All jobs and Testbeds are released once the scheduling is deleted.

The Schedulings Resource may be modified at any time, moving the scheduling from the AwaitingCompletion state back to the Core submission Loop.


## External-Scheduler-Interface

The external-scheduler-facing interface is currently part of the Scheduling Reconciler. Most of the functionality has already been discussed in the previous chapter. 

The Interface is only a very thin wrapper around the Kubernetes API. In its current state, it is only used to hide unnecessary complexity from the external scheduler and offers are more simplistic API to interact with Kubernetes and the External-Scheduler-Interface components.

A reference implementation for using the API is part of the Manual Schedulers website. 