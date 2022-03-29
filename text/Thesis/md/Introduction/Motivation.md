The current population is producing more and more data. This creates an excellent opportunity for many businesses. Businesses willing to profit from collected data to improve their sales strategies have to collect and store not just gigabytes but upwards to petabytes of data[@marr2017really]. With more affordable storage costs, companies are even less likely to toss away potential valuable data, creating so-called data lakes[@miloslavskaya2016big].

Collecting data is only the first step. It takes many stages of processing, through aggregation and filtering, to extract any meaningful information. Usually, the sheer mass of collected data makes it not very useful to begin with.

Unfortunately, when working with petabytes of data, it is no longer feasible to work on a single machine. Especially when dealing with a stream of data produced by a production system and the information collected from yesterday's data is required the next day, ideally immediately.

Scaling a single machine's resources to meet the demand is also not feasible. It is either very expansive or might just straight up not be possible[@agrawal2010data]. On the other hand, cheap commodity hardware allows the scaling of resources across multiple machines is much cheaper than investing in high-end hardware or even supercomputers[@weiss2007computing].

The complexity of dealing with a distributed system can be reduced using the abstraction of a cluster. A **cluster resource manager** is used, where a system of multiple machines forms a single coherent cluster that can be given tasks to.

Stream processing of data across such a cluster can carry out using stream or batch processing frameworks, such as Apache Spark or Apache Flink. These frameworks already implement the quirks of dealing with distributed systems and thus hide the complexity.

The problem is that multiple batch jobs running on a single cluster need resources that need to be allocated across the cluster. While resource allocation is the task of the cluster resource manager, the manager usually does not know how to allocate its resources optimal and often requires the user to specify the resources allocated per job. This usually leads to either too few resources being allocated per job, starving jobs and increasing the runtime, or over-committing resources, thus leaving resources in the cluster unused[@delimitrou2014quasar].

Another problem that arises is the fact that even though the reoccurring nature of batch jobs, not all batch jobs use the same amount of resources. Some are more computationally intensive and require more time on the CPU, while others are more memory intensive and require more or faster access to the machine's memory. Others are heavy on the I/O usage and use most of the system's disk or network devices. This shows up in vastly different job runtime (also total runtime) depending on the scheduling of batch applications across the cluster[@thamsen2021mary].

Finding an intelligent scheduling algorithm that can identify reoccurring jobs and estimate their resource usage based on collected metrics and thus create optimal scheduling is not an easy task. It also requires a lot of setup when dealing with a cluster resource manager.
