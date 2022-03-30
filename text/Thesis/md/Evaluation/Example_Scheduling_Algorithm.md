The Interfaces usability is evaluated with an exemplary implementation of a non-trivial scheduling algorithm. Since the Interfaces can manage multiple Testbeds inside the same cluster, a Profiling-Scheduler approach is chosen to highlight some of the Interface's features. 

The example scheduling algorithm is used on a 5 Node cluster running inside the Google Cloud Platforms Kubernetes Engine (GKE). During development, smaller Nodes with a single vCPU and 4Gi of Memory were sufficient, but for the final evaluation, Nodes were doubled in capacity.

![Architecture of the Example Profiler-Scheduler](graphics/evaluation_example_scheduler_arch.pdf)

Two Testbeds are created using the Testbed CRD, the Profiling Testbed, and the Testbed for the actual execution of Jobs. The slot size was chosen depending on the resources available. For the final evaluation, a slot size of 750m CPU and 3Gi Memory leaves enough resources available for the cluster's control plane, managing Pods (Driver, JobManager), and the Operators running inside the cluster.

Contrary to the Scheduler Thread, which only creates a scheduling if requested (via stdin), the Profiler Thread runs at all times, updating and refining the co-location Matrix by choosing job pairings with the least data points. To test the stability of the system, the scheduling thread was later modified to create schedulings by choosing jobs at random and running continuously in a loop.



|                  | flink-wc       | spark-pi       | spark-pi-long    | flink-wc2       | spark-crawler |
| ----------------:|:--------------:|:--------------:|:----------------:|:---------------:|:-------------:| 
| flink-wc         |       /        |     36 (5)     |     28 (11)      |     24 (16)     |    34 (11)    | 
| spark-pi         |     70 (4)     |       /        |      93 (4)      |     68 (4)      |    94 (4)     | 
| spark-pi-long    |    444 (4)     |    480 (6)     |        /         |     450 (3)     |    493 (3)    | 
| flink-wc2        |    24 (16)     |     36 (5)     |     30 (11)      |        /        |    25 (10)    | 
| spark-crawler    |    173 (3)     |    203 (5)     |     211 (3)      |     171 (3)     |       /       | 

Table: Co-Location matrix calculated by the profiler. \label{tbl:coLocationMatrix}


Table \ref{tbl:coLocationMatrix} shows the co-location matrix that the profiling scheduler calculates by iterating over the available job pairings and measuring their runtime. Each row describes how the runtime of the job changes if it is co-located with another job. 

Co-locations are described as a simple runtime in seconds. The Profiler creates a cumulative moving average for each job pairing.

Jobs cannot be paired with themself since the acquire/release mechanism only allows a single execution per job at a time. In theory, Jobs can be co-located with itself by deploying the application with a replication of two, but this could not be directly compared to co-location with a different job, as work done is split between both instances.

The matrix is not symmetrical because the runtime of each job is used, not the runtime of both jobs (or the runtime of the complete scheduling). The runtime of the scheduling is the time after acquisition until all jobs have been completed. The BatchJob Operator only tracks the application's time inside the running state. This approach was chosen because applications may have vastly different startup times, which will become insignificant for long-running jobs.

The Scheduling Thread is the traditional scheduler. Given a list of Jobs, the Scheduling Thread tries to find optimal scheduling regarding total runtime. The scheduler takes a greedy approach choosing the co-located job based on the job with the shortest runtime to keep the evaluation simple. Replication of each job is selected based on the number of slots (*Replication = NumberOfSlots / NumberOfJobs*). Empty slots are again greedily filled with jobs suited best for co-location, not allowing a job to be chosen more than once.

Both the Profiler and the Scheduler run in parallel. If any of them cannot acquire their jobs, the scheduling will wait until they become available.

While neither the BatchJobs used during the evaluation nor the Scheduling algorithm are particularly sophisticated, the evaluation showed that the Extern-Scheduler-Interface could guarantee the placement of distributed dataflow applications to the cluster. The actual implementation of the scheduling algorithm is straightforward and can be found in the Appendix.