## Batch Job Operator

The Batch Job Operator is composed of the Batch Job Reconciler and the Batch Job CRD. The Reconciler is listing for changes, regarding the Batch Jobs CRs and Applications CRs, which are managed by the Spark Operator and the Flink Operator.

The Batch Job Operator knows, given the Batch Job CRs specification, how to construct the corresponding Application. This was made easy due to the fact, that the Batch Job CR is only a thin wrapper around either the Spark or the Flink specification. In addition to the Spark and and Flink CR it also, may contain additional Information, that previous invocations of the External-Scheduler have stored.

Currently the Batch Job CR, does only contain a partial application specific specification. Although it would not matter, if a user would submit a fully specified Flink or Spark Application, the Operator would overwrite, most of the Driver/Executor Pod specific configuration and replication configuration. The Aim of the Batch Job CR is to allow a user to specify only the Required Components, like Application Image containing the actual Application and its arguments like a DateSet or where to find it (e.g. using HDFS).

![StateMachine](graphics/batch_job_state_machine.pdf)

The BatchJob Reconciler is implemented as a Nested State Machine with anonymous SubStates. The Approach was chosen, as it creates more comprehensible Software, which can be split easier into Components, and handle Edge Case by Design. More Information about the Design of the State Machine is outlined in the separate Section.

Initially BatchJobs submitted to the Cluster remain in the ReadyState. While in the ReadyState the BatchJob Operator, will not do anything. A *Scheduling* can claim a BatchJob, in which case the BatchJob CR will move into the InQueueState until the *Scheduling* instructs the Batch Job Operator to create the Application and track its lifecycle.

Communication between the Batch Job Reconciler and the Scheduling Reconciler is done via the Kubernetes label mechanism. The *Scheduling* Reconciler places the **ACTIVE_SCHEDULING** labels on any Batch Job CR it will use during the Scheduling. This mechanism ensures that only one Scheduling at a time can use the Batch Job, on the flip side a *Scheduling* can claim multiple *Batch Jobs*. If the active scheduling releases the job, by removing the label, the Batch Job moves back into the Ready State, this can happen could happen at any time, and may even cause any created application to be removed.

Once a Batch Job is in the InQueue State, the reconciler waits for the creation request issued by the *Scheduling* Reconciler. The requests is again done using **APPLICATION_CREATION_REQUEST** labels and specifies desired replication and *Slots* the Application will use.

**NOTE:** Kubernetes Label mechanism is chosen, because it ensures, that only one scheduling at a time should have control over the Batch Job. A Scheduling will set the Labels on a batch job only if no other has claimed the job. If it detects that it is no longer the active scheduling, which might happen if the same is job claimed by a different scheduling at the same time, it will not proceed the scheduling, but remove all other labels from other jobs to prevent deadlocks. More details how the claiming mechanism was design are in the Scheduling Reconciler Section.

When configuring the Application, to be created by the corresponding Operator, there are two types of configuration. Configuration can either be:

- Persisted inside the Batch Job CR, which is used on every invocation of the Application. This includes the Applications Image and arguments, like the data set

- Scheduling dependent. These configuration can not be stored inside the CR and need to be supplied with the creation request.

After a Batch Job was requested to create the Application, application specific logic is executed. In any case, the actual steps for deploying the applications to the cluster is done by the Applications Operator (Flink Operator[@FlinkOperator] or Spark Operator[@SparkOperator]). The Batch Job Reconciler only instructs the Application Operators, with configurations for the Executor/TaskManager Pods that will be created to be identifiable by the Scheduler Extender.

When creating the application, the following aspects are configures for the Executor/TaskManager Pods:

- **Resource Requests**: The Container resources are specified by the Testbeds Slots, in order for the pods to fit inside the Slots they need the correct Resource Request. (Currently only CPU and Memory)

- **Slot IDs**: The scheduling (or the external scheduler) decides which slots are used by which job. In order for the Executor/TaskManager Pods to be placed into the correct slot (technically the correct Node), Pods need to be made identifiable by the Scheduler Extender.

- **Replication**: The Number of Executor/TaskManager Pods depends on the Number of Slots that will be used for the Application.

- **Priority Class**: Application pods need a *PriorityClass* otherwise, preemption will not be triggered by the Kubernetes Scheduler.

- **Scheduler Name**: Application pods need a *SchedulerName* otherwise, the default KubeScheduler will handle the scheduling and thus ignore the Scheduling Extender.

Configuration of **Resource Requests**, **Replication** are straight forward, as both the Spark and Flink Operator expose these via their respective CRDs. The Spark Operator actually, exposes the complete PodSpec for both driver and executor pods, whereas the Flink Operator only exposes a few PodSpec attributes. The Flink Operator had to be extended with the missing configurations. This way **Resource Requests**, **Replication**, **Priority Class**, **Scheduler Name** are configured.

The difference between any of the mentioned above configurations and the **Slot IDs** is that the Application Operators only allow (rightfully so) to specify a single pod spec. The reason for this, is that the Executor/TaskManager Pods are controlled by a Stateful Set, which scales up to the desired Replication. However the above mentioned configurations, are valid for all pods, but *Slot IDs* should be different for all pods.

This issue can be circumvented by the leaving the final decision of which pod goes into which slot to the Extender and submitting a list of SlotIDs to the extender. The Extender can that decide which pod goes into which slot. Pods are configured with a affinity of the combined set of nodes where the slots reside on.

![Affinities](graphics/affinity.png){height=25%}

Once the Application was Created the Job moves into the Submission State, and resides there until all Pods of the Application where scheduled, at which point it moves on into the Running State. At this Point the underlying Applications state is monitored, until it moves into the Application Specific Completed state, (currently for Spark: *Completed* and for Flink: *Stopped*). During the implementation, scenarios were encountered, in which the BatchJob reconciler was not running, and once restarted found Applications in a completed state without passing the scheduling, submission and running state. In Order to prevent any tight coupling none of these transitions are required, to be considered a successful execution.

The BatchJob reconciler tracks the time an application ran, by creating timestamps once the application, started running and its completion.
More details about how the Scheduling Extender works are inside the Extender Section. 

## Slot Operator

The Slot Operator is composed of the Reconciler Loop and the Slots CRD. The Slots CR is supposed to model a Test Bed of slots located in a Cluster of Machines. Slots can have specified Resources. While no application is running inside a slot it is considered *free*. In Order to reserve resources in the cluster, and thus guarantee applications, supposed to be deployed inside a *free* slot to actually get the resources, the Slot Operator needs to:

- **Reserve Resources** by using so-called Ghost Pods inside the cluster, that specify a resource request and thus reserve the resources

- **Preempt** Ghost Pods for pods that wants to be deployed inside a slot

The Slots Reconciler listens to changes to the Slots CR, and the current cluster situation. It makes sure that always the correct number of pods, with the specified resource requests, are deployed onto the cluster. The Slots CR is composed of the following configurations:

- Label Name to Identify any nodes that are part of the Test Bed. Only the Label Name is specified not a specific value. The value is later used to create a distinct order of slots in the cluster.

- Number of Slots per Node

- Resource Request per Slot

Given the Test Beds specification, the reconciler listens for all changes to nodes **with** the specified label, but also to all nodes **without** any label in case the label was removed and the test bed needs to be resized. Further it also listens to changes to any pod which is part of the Test Bed.

The typical Reconciliation Loop works as following:

- Fetch the current cluster situation

- Calculate the desired cluster situation

- Find the difference. Either delete undesired Pods, or create desired Pods


Fetch the current cluster situation, by fetching all pods with the **SLOT** label. Pods are then grouped by their Node, thus creating a list of pods per node. On the Flip side we calculate the desired state by modeling pods for every slot and also group the by node. When comparing pods we consider them equal, if they reside on the same node and have the same resource request. 

![New Pods need to be Created](graphics/TestBed-ObservedAndDesired.pdf)
![ResourcePerSlot Change: New Pods need to be Created](graphics/TestBed-ObservedAndDesired(1).pdf)
![Node Change: Pods need to be Deleted](graphics/TestBed-ObservedAndDesired(2).pdf)
![Desired State: Pods need to be Deleted](graphics/TestBed-ObservedAndDesired(3).pdf)


**Note**: During early implementation it was assumed that there can only be on Test-Bed Active at a Time. While technically that may not be necessary anymore, since every component using the Test Bed specifies its exact name and namespace. Some parts of the Slots reconciler might have been implemented with the initial assumption in mind. E.g. Verifying that enough resources are available on every node, does not account for possible multiple active Test Beds on a node.


**Note**: Slots are positioned in a round-robin fashion

**Node**: The actual position of Slots on a Node does not matter, slots on are Node are only a logical abstraction.


