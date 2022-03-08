- Abstraction for Systems with many machines

*(TODO: Super Computer/HPC)*
- Super Computer with high-end hardware
- Scale Work across multiple machines benefitting from cheap commodity hardware
- Coarse Grain: Static partitioning of Machines across different workloads is enough

*(What is MapReduce, maybe put into Big Data Stream Processing)*

- Hadoop is one of Many Open-Source MapReduce implementations
*(TODO: Frameworks)*
Cluster computing using commodity hardware was driven by the need to keep up with the explosion of data.
The Initial Version of Hadoop was focused on Running MapReduce Jobs to process a Web Crawl (*YARN Paper*). Despite the initial focus, Hadoop was widely adopted evolved to a state where it was no longer used with its initial target in mind. Wide adoptions have shown some of the weaknesses in Hadoops architecture:
- Tight Coupling between the MapReduce Programming model and Cluster Management
- Centralized Handling of Jobs will prevent Hadoop from Scaling

![Static Partitioning](graphics/static_partitioning.png){width=25%, height=25%}
![Dynamic Partitioning](graphics/dynamic_partitioning.png){width=25%, height=25%}

The tight coupling leads Hadoop users with different applications to abuse the MapReduce Programming model to benefit from cluster management and be left with a suboptimal solution. A typical pattern was to submit 'map-only' jobs that act as arbitrary software running on top of the resource manager. [@10.1145/2523616.2523633]
The other solution was to create new Frameworks. This caused the invention of many frameworks that aim to solve distributed computation on a cluster for many different kinds of applications [@hindman2011mesos].
Frameworks tend to be strongly tailored to simplify solving specific problems on a cluster of computers and thus speed up the exploration of data. It was expected for many more frameworks to be created, as none of them will offer an optimal solution for all applications.[@hindman2011mesos]

Initially, Frameworks like MapReduce created and managed the cluster, which only allowed a single Application across many machines—running only a single application across a cluster of nodes led to underutilization of the cluster's resources. The Next generation of Hadoop allowed it to build ad-hoc clusters, using Torque [@1558641] and Maui, on a shared pool of hardware. Hadoop on Demand (HoD) allowed users to submit their jobs to the cluster, estimating how many machines are required to fulfill the job. Torque would enqueue the job and reserve enough machines once they become available. Torque/Maui would then start the Hadoop Master and the Slaves, subsequently spawn the Task and JobTracker that make up a MapReduce Application. Once all Tasks are finished, the acquired machines are released back to the shared resource pool.
This can create potentially wasteful scenarios, where only a single Reduce Task is left, but many machines are still reserved for the cluster. Usually, Resources requirements are overestimated and thus leaves many clusters resources unused.

With the Introduction of Hadoop 3, the MapReduce Framework was split into the dedicated Resource Manager YARN and MapReduce itself. Now MapReduce was no longer running across a Cluster, but it was running on top of YARN, who manages the cluster beneath. This allows MapReduce to run multiple Jobs across the same YARN Cluster, but more importantly, it also allows other Frameworks to run on top of YARN. This moved a lot of complexity away from MapReduce and allowed the framework to only specialize on the MapReduce Programming Model rather than managing a cluster. This finally allows different Programming Models for Machine Learning tasks that tend to perform worse on the MapReduce Programming model [@zaharia2010spark] to run on top of the same cluster as other MapReduce Jobs.

Using the ResourceManager YARN allows for a more fine-grain partitioning of the cluster resources. Previously a static partitioning of the clusters resources was done to ensure that a specific application could use a particular number of machines. Many applications can significantly benefit from being scaled out across the cluster rather than being limited to only a few machines.
 - Fault tolerance: Frameworks use replication to be resilient in the case of machine failure. Having many of the replicas across a small number of machines defeats the purpose
 - Resource Utilization: Frameworks can scale dynamically and thus use resources that are not currently used by other Applications. This allows applications to scale out rapidly once new nodes become available
 - Data Locality: Usually, the data to work on is shared across a cluster. Many applications are likely to work on the same set of data. Having applications sitting on only a few nodes, but the data to be shared across the complete cluster leads to a lousy data locality. Many unnecessary I/O needs to be performed to move data across the cluster.

Fine-grain partitioning can be achieved using containerization, where previously applications were deployed in VMs, which would host a full Operating System. Many containers can be deployed on the same VM (or physical machine) and share the same Operating System Kernel. The hosting Operating makes sure Applications running inside containers are isolated by limiting their resources and access to the underlying Operating System.

Before Hadoop 3 with YARN was published, an Alternative Cluster Manager Mesos was publicized. Like YARN, Mesos allowed a more fine granular sharing of resources using containerization.
The key difference between YARN and Mesos is how resources are scheduled to frameworks running inside the cluster. YARN offers a resource request mechanism, where applications can request the resources they want to use (*TODO: fact check*), and Mesos, on the other hand, offers frameworks resources that they can use. This allows frameworks to decide better which of the resources may be more beneficial. This enables Mesos to pass the resource-intensive task of online scheduling to the frameworks and improve its scalability.


*(TODO: Kubernetes)*
Kubernetes was initially developed by Google and released after multiple years of intern usage. Kubernetes was quickly adopted and has become the defacto standard for managing a cluster of machines.
Kubernetes offers a descriptive way of managing resources in a cluster where manifests describing the cluster's desired state are stored inside a distributed key-value store etcd. Controllers are running inside the cluster to monitor these manifests and do the required actions to bring the cluster into the desired state. 
Working with manifest abstracts away many of the problems that arise when deploying Applications to a cluster. Usually, an Operations Team was required to manage applications across the cluster. 
With Kubernetes offering the required building blocks and the mechanism of a control-loop, the operator pattern in combination with Custom Resource Definitions is commonly used to extend Kubernetes functionalities.


### Notes (Ignore):
- HPC Super Computer / Resources
- Kubernetes Section is definitely not complete


- Explain what a Cluster Resource Manager is doing
    - Abstraction of using a single cluster as a single Machine
    - Managing given resources making it scalable by adding more machines
- Show what are the differences between YARN and Kubernetes
    - YARN: Emerging from Hadoop was design to Work with Batch Applications
    - Kubernetes: All Round Cluster Manager, with a Big Community

*(TODO: Cloud Computing)*
- Coarse Grain approach is no longer feasible
- Applications may scale up or down, for and a partitioning of the cluster has to be done dynamically
- Containerization, allows to run many different applications on the same machine without much overhead, like creating a new virtual machine
- Different Applications may share same Node to increase overall utilizations of resources
- Applications benefit from data locality, where creating application on a node that already contains the data, will not use additional I/O Resources
- Different Applications are likely to work on the same data


Mesos and Kubernetes
- etcd vs ZooKeeper
- Mesos Resource Offer leave scheduling to Framework
- Kubernetes follows a descriptiv approach, where the user of the cluster describes a state that the cluster should be in and the resource manager
exectues the required actions. This makes Kubernetes very extensible.
- Kubernetes builds a virtual network across nodes
- Kubernetes general purpose


*YARN Paper*
- Hadoop thightly focused on Web Crawling for Hadoop at Yahoo!
- Broad Adoption streched intial focus
- Thight Coupling of MapReduce Programming Model and Resource Manager
- Centralized handling of Jobs control flow from the JobTracker

- MapReduce Programming model was abused for other purposes then it was initially design
- Common Pattern of Map-Only that was used for Forking?-Web-Services and Gang-Scheduled Computation. In general Developer came up with clever solutions to run all kinds of software on top of hadoop
- Misuse and Broad adoption exposed many substantial issues with Hadoops archicture and implementation
- YARN moves Hadoop pasts its original incarnation, by breaking up the monolithic architecture of Hadoop, splitting it into a Resource Manager from the programming model
- YARN delegates many scheduling-related functions to per-job components
- This makes MapReduce just one of many applications that can run on top of YARN
- Allows for great choice of Programming Models using different Frameworks
- Programming Frameworks running on top of YARN can manage their intra-application communication, execution flow and dynamic optimization by them self, unlock performance benefits

### Historical
- Yahoo! WebMap
- Needs to be scalable 
- Ad-hoc clusters
 - Initially user bring up a handful of nodes, load their data into HDFS, write a MapReduce job, then tear it down
 - Hadoop becomes more fault tolerant and persistent HDFS would become the norm
 - Operators just upload potential interesting data into the the HDFS, attracting analysts. --> Mulit-Tenancy of HDFS
 - To address Multi Tenancy HoD was deployed, using Torque and Maui to allocate Hadoop Cluster on a shared pool of Hardware
 - User submit Job, with required amount of rescoures to torque, which then waits until enough resources are avaible
 - Torque would start the Hadoop Leader, which then interacts with Torque and Maui to create the Hadoop Slaves, which will then Spawn the TaskTracker + JobTracker
 - Job Tracker would then accept a series of Jobs
 - User can then release the Cluster. The System will then return logs to the user and return nodes to the cluster
 - HoD allows slightly older version of Hadoop to be used
- HoD shortcomming:
 - Torque did not account for data locality
 - Bad resource utilization, last task of job may block hundreds of machines, Cluster were not resized between jobs, Users usually overestimate the number of nodes required
- Shared Cluster
 - Resource Granularity was to coarse 
 - JobTracker does not scale as HDFS, JobTracker failure was a complete failure of all jobs
 - 
