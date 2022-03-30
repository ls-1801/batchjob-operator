HPC supercomputers are commonly built with expensive hardware and tightly coupled operating systems, allowing parallel applications to take advantage of resources available on the supercomputer. supercomputers are carefully planned with software designed for specific purposes. This usually meant that multiple applications running on a supercomputer would have resources statically partitioned. Coarse grain partitioning introduces inefficient use of hardware once part of the statically partitioned resources is no longer in use.

![Static Partitioning](graphics/static_partitioning.png){width=40%} \ ![Dynamic Partitioning](graphics/dynamic_partitioning.png){width=60%}

\begin{figure}[!h]
\begin{subfigure}[t]{0.4\textwidth}
\caption{Static partitioning\label{fig:staticPartitioning}}
\end{subfigure}
\hfill
\begin{subfigure}[t]{0.6\textwidth}
\caption{Fine grain partitioning using containers\label{fig:fineGrainPartitioning}}
\end{subfigure}
\end{figure}


With the emergence of distributed dataflow applications, like Hadoop MapReduce [@dittrich2012efficient], the need for a more fine-grain partitioning of cluster resources came along. The move away from supercomputers to cheap commodity hardware meant that clusters could be scaled up or down easily, where previously careful planning was required. Managing a dynamic system of potentially thousands of machines can not be done in a manual fashion. A cluster resource manager is required to build the abstraction of a single cohesive cluster that can be tasked with jobs.

Hadoop is one of many open-source MapReduce implementations. Cluster computing using commodity hardware was driven by the need to keep up with the explosion of data.
The initial version of Hadoop was focused on running MapReduce jobs to process a web crawl [@10.1145/2523616.2523633]. Despite the initial focus, Hadoop was widely adopted evolved to a state where it was no longer used with its initial target in mind. Wide adoptions have shown some of the weaknesses in Hadoops architecture:
- tight coupling between the MapReduce programming model and cluster management
- centralized Handling of jobs will prevent Hadoop from scaling



The tight coupling leads Hadoop users with different applications to abuse the MapReduce programming model to benefit from cluster management and be left with a suboptimal solution. A typical pattern was to submit 'map-only' jobs that act as arbitrary software running on top of the resource manager. [@10.1145/2523616.2523633]
The other solution was to create new frameworks. This caused the invention of many frameworks that aim to solve distributed computation on a cluster for many different kinds of applications [@hindman2011mesos].
Frameworks tend to be strongly tailored to simplify solving specific problems on a cluster of computers and thus speed up the exploration of data. It was expected for many more frameworks to be created, as none of them will offer an optimal solution for all applications [@hindman2011mesos].

Initially, frameworks like MapReduce created and managed the cluster, which only allowed a single application across many machines—running only a single application across a cluster of machines led to the underutilization of the cluster's resources. The next generation of Hadoop allowed it to build ad-hoc clusters, using Torque [@1558641] and Maui, on a shared pool of hardware. Hadoop on Demand (HoD) allowed users to submit their jobs to the cluster, estimating how many machines are required to fulfill the job. Torque would enqueue the job and reserve enough machines once they become available. Torque/Maui would then start the Hadoop master and its slaves, subsequently spawning the Task and JobTracker that make up a MapReduce application. Once all tasks are finished, the acquired machines are released back to the shared resource pool.
This can create potentially wasteful scenarios, where only a single reduce task is left, but many machines are still reserved for the cluster. Usually, resources requirements are overestimated and thus leaves many clusters resources unused [@delimitrou2014quasar].

With the introduction of Hadoop 3, the MapReduce Framework was split into the dedicated resource manager YARN and MapReduce itself. MapReduce was no longer running across a cluster, but it was running on top of YARN, who manages the cluster beneath. This allows MapReduce to run multiple jobs across the same YARN cluster, but more importantly, it also allows other frameworks to run on top of YARN. This moved a lot of complexity away from MapReduce and allowed the framework to only specialize on the MapReduce programming model rather than managing a cluster. This finally allows different programming models for machine learning tasks that tend to perform worse on the MapReduce programming model [@zaharia2010spark] to run on top of the same cluster as other MapReduce jobs.

Using the resource manager YARN allows for a more fine-grain partitioning of the cluster resources. Previously a static partitioning of the clusters resources was done to ensure that a specific application could use a particular number of machines. Many applications can significantly benefit from being scaled out across the cluster rather than being limited to only a few machines.
 - Fault tolerance: frameworks use replication to be resilient in the case of machine failure. Having many of the replicas across a small number of machines defeats the purpose
 - Resource utilization: frameworks can scale dynamically and thus use resources that are not currently used by other applications. This allows applications to scale out rapidly once new machines become available
 - Data-locality: usually, the data to work on is shared across a cluster. Many applications are likely to work on the same set of data. Having applications sitting on only a few machines, but the data to be shared across the complete cluster leads to a lousy data locality. Many unnecessary I/O needs to be performed to move data across the cluster.

Fine-grain partitioning can be achieved using containerization, where previously applications were deployed in VMs, which would host a full operating system. Many containers can be deployed on the same VM (or physical machine) and share the same operating system kernel. The hosting operating system makes sure applications running inside containers are isolated by limiting their resources and access to the underlying operating system.

Before Hadoop 3 with YARN was published, an alternative cluster manager Mesos was publicized. Like YARN, Mesos allowed a more fine granular sharing of resources using containerization.
The key difference between YARN and Mesos is how resources are scheduled to frameworks running inside the cluster. YARN offers a resource request mechanism, where applications can request the resources they want to use, and Mesos, on the other hand, offers frameworks resources that they can use. This allows frameworks to decide better which of the resources may be more beneficial. This enables Mesos to pass the resource-intensive task of online scheduling to the frameworks and improve its scalability.

The concept of containerization, which offers a lightweight alternative to traditional VMs, has been internally used by Google many years before YARN and Mesos. Google has used container orchestration systems internally for multiple years before releasing the open-source project Kubernetes [@burns2016borg]. Kubernetes was quickly adopted and has become the defacto standard for managing a cluster of machines. Kubernetes offers a descriptive way of managing resources in a cluster where manifests describing the cluster's desired state are stored inside a distributed key-value store [@grzesik2019evaluation] etcd. Controllers are running inside the cluster to monitor these manifests and do the required actions to bring the cluster into the desired state. With Kubernetes offering building blocks and the mechanism of a control loop, the Operator pattern in combination with Custom Resource Definitions (CRDs) is commonly used to extend Kubernetes functionalities. Where Hadoop's YARN was focusing on distributed dataflow applications, Kubernetes enables developers of all kinds of applications the scalability and resilience of the cloud.

