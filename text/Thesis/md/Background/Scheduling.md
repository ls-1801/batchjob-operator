In general, scheduling is the process of assigning resources to a task. This includes the question:

1. Should any resources be allocated for the task at all
2. At which point in time should resource be allocated
3. How many resources should be allocated
4. Which of the available resources should be allocated


*(TODO: Where scheduling is necessary)*

Scheduling is essential for Operating Systems that need to decide which process should get CPU time and which processes may need to wait to continue computation. In the case of multiple CPU, a decision has to be made which CPU should carry out the computation.
The Operating System is not just concerned with CPU-Resources, but also I/O Device resources. Some devices may not work under concurrent usage and require synchronization. Who is allowed to access it?


*(TODO: What scheduling policies/strategies exist, and what are they aiming to optimize)*

In some cases, a simple FIFO scheduling that works on tasks in order there were submitted produces acceptable results.
*(TODO: Scheduling is used to optimize for Deadlines/Throughput/FastResponseTime)*
Scheduling depends on a goal. Some Algorithms aim to find the optimal schedule to respect any given deadlines. Whereas some distinguish between Soft and Hard Deadlines, where ideally you would not want to miss any deadlines, occasionally missing Soft deadlines to guarantee Hard Deadlines are met is acceptable. In general, finding a single best schedule that allows resources to be allocated optimally is not possible.
Scheduling for a fast response time or throughput might prefer shorter tasks to be run, when possible, and might starve longer running tasks for a long time before progress can be made.


*(TODO: Preemptive)*

A Scheduling algorithm might allow preemption, where the currently active task could be preempted for another task to become active. Some scheduling algorithms account for the potential overhead of preempting the current task (like a context switch).


The higher up the Stack *(TODO: stack = single process -> os -> vms -> Distributed Systems)*, more and more potential schedules become possible. It seems a wise choice for the scheduling to be handled in their respective stack layers.

The Question of Scheduling in a Distributed System is now the question of which machines resources should be used for which task. Here Scheduling algorithms need to pay attention to the characteristics of a Distributed System:
1. Potential heterogeneity of the system, with machines of different Hardware and different Operating Systems or Software
2. Spontaneously adding and removing resources of the Cluster
3. Interference between Applications residing on the same machine, same rack, same network switch, etc. (CO-Location)

While some of these factors can be controlled, different algorithms can be chosen for various use cases.

*(TODO: Scheduling in Stream Processing: DAG Level Scheduling / Container Level Scheduling)

In Stream-Processing, we deal with a multitude of different levels of scheduling. Stream Processing Frameworks build the DAG based on the job submitted. The initial DAG breaks done the job into their respective Map and Reduce Operations. These Operations will be broken down into smaller Tasks based on the Partitioning of Data. Finally, the Tasks may be executed on an arbitrary number of machines (technically not machines, but processes). Optimizing the schedule of tasks to a machine will be called DAG-Level scheduling and may now also include factors like Data-Locality.


*(TODO: Scheduling on the Cluster Level: The Interesting Topic of this Thesis)*

Moving Up one level Higher in the Stack, we are concerned with running multiple Jobs inside the same cluster, and a decision needs to be made which job can spawn their executor on which nodes. Executors are packaged on Containers. The containers are isolated, so they can not access each other. Unfortunately, isolating processes from each other in a container forces the underlying machine to need to know how much of the system's CPU and memory each container should have. 



### TODO:
- Explain what Scheduling is
- Different kind of scheduling
    - DAG Scheduling done by Spark (Not what this thesis is about)
    - POD Scheduling done by the Cluster Resource Manager
        - Co-Location
        - Packing
- Explain why Scheduling is Important
    - Co-Location Problem
    - Low Resource Usage (Graph from Google)
    - Results from Hugo/Mary Paper


### Open:
- Should maybe start a bit less specific about this Thesis and find more Information about Scheduling in general or is it fine if the Background section starts of general and tailors towards the topic of my thesis?