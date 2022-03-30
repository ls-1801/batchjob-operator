In general, scheduling is the process of assigning resources to a task. This includes the question:

1. Should any resources be allocated for the task at all?
2. At which point in time should resource be allocated?
3. How many resources should be allocated?
4. Which of the available resources should be allocated?

Scheduling is essential for Operating Systems that need to decide which process should get CPU time and which processes may need to wait to continue computation. In the case of multiple CPUs, a decision has to be made on which CPU should carry out the computation.
The operating system is not just concerned with CPU-Resources, but also I/O Device resources. Some devices may not work under concurrent usage and require synchronization.

In some cases, a simple FIFO scheduling that works on tasks in order there were submitted produces acceptable results.
Scheduling depends on a goal. Some algorithms aim to find the optimal schedule to respect any given deadlines. Whereas some distinguish between soft and hard deadlines, where ideally no deadlines would be missed at all, occasionally missing soft deadlines to guarantee hard deadlines are met is acceptable. In general, finding a single best schedule that allows resources to be allocated optimally is not possible.
scheduling for a fast response time or throughput might prefer shorter tasks to be run, when possible, and might starve longer running tasks for a long time before progress can be made.

Common goals of scheduling are [@hpcWikiScheduling]:

- Minimizing the time between submitting a task and finishing the task. A task should not stay in the queue for a long time and start running soon after submission.
- Maximizing resource utilization. Resources available to the scheduler should be used, even if that means skipping tasks waiting in the queue.
- Maximizing throughput. Finishing as many jobs as possible may starve longer running jobs in favor of multiple shorter running jobs.

To achieve their goals scheduling algorithm might allow preemption, where the currently active task could be preempted for another task to become active. Some scheduling algorithms account for the potential overhead of preempting the current task (like a context switch).

Scheduling can be applied at multiple levels of the software stack, like scheduling processes at the operating system level, virtual machines at the hypervisor level, or scheduling tasks across many applications running on many machines at the cluster level. The higher up the stack, the more and more potential schedules become possible. It is a balancing act of complexity and performance. At the highest level, the scheduler has the most knowledge about the systems and could come up with optimal scheduling. However, it seems to be a wise choice for scheduling to be handled in their respective stack layer since a lot of complexity can be encapsulated inside each layer. Allowing the top level scheduler to micromanage the complete stack creates an explosion of complexity.

For this work, the scope of scheduling is limited to batch scheduling of Jobs in a cluster.

The Question of Scheduling in a Distributed System is now the question of which machines resources should be used for which job. In batch scheduling algorithms need to pay attention to the characteristics of a Distributed System:

1. Potential heterogeneity of the system, with machines of different Hardware and different Operating Systems or Software
2. Spontaneously adding and removing resources of the Cluster
3. Interference between Applications residing on the same machine, same rack, same network switch, etc. (CO-Location)

While some of these factors can be controlled, different algorithms can be chosen for various use cases.

In distributed dataflow applications, we deal with considerable different levels of scheduling. At the framework level, applications build a DAG-based execution plan based on the job submitted. The initial DAG breaks done the job into their respective Map and Reduce Operations. These Operations will be broken down further into smaller Tasks based on the Partitioning of Data. Finally, Tasks are executed on an arbitrary number of processes across different machines. Optimizing the schedule of tasks to an executor process will be called DAG-Level scheduling and may now also include factors like Data-Locality.

Moving Up one Level Higher in the stack, we are concerned with running multiple Jobs inside the same cluster, and a decision needs to be made which job can spawn their executor on which Nodes. Executors are packaged on Containers. The containers are isolated so that they can not access each other.

A more profound introduction into the specifics of scheduling in Kubernetes is inside the Scheduling in Kubernetes chapter.

Cluster resource manager like Mesos [@hindman2011mesos] allows scheduling on multiple levels, with Resource Offers. The top-level Mesos scheduler finds multiple candidates and offers them to the DAG-Level scheduler framework. The potential for cross-level scheduling would allow the Top-level scheduler to respect data-locality constraints for the DAG-level scheduler.