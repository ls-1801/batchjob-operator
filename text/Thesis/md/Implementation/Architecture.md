In this section, all components that make up the Interface are introduced. Here an Architectural overview is presented, and interactions between components are discussed.

The current implementation of the Scheduler Interface consists of 5 Components that will be introduced in this section but discussed in more detail in the Operator Section.

![Components](graphics/architecture_components.png){width=50%, height=50%}

The five components consist of three Reconciler or Control-Loops, the Batch-Job Reconciler, the Slots Reconciler, and the Scheduling Reconciler. The architecture also uses an Extender and, finally, the External-Scheduler facing Web-API.

The Interface that is visible to an External Scheduler is supposed to be simple and only allows the querying of the current cluster situation, information about previous scheduling, and submission of new schedulings. Further, more concrete information, like node metrics, can be queried from a Metric Provider that is commonly deployed along with the Cluster.

Additionally to the three Reconciler, three Custom Resource Definitions (CRDs) are created.

- *Batch Job* represents an Abstract Batch Job Application and can store information the external scheduler may want to remember for future invocations

* *Slots*represent a testbed of guaranteed resources available for a *Scheduling*. A *Slots* Custom Resource is a collection of slots across the clusters node that are referenced in a *Scheduling*

- *Scheduling*represents the decision done by the External-Scheduler. A *Scheduling* maps multiple *Batch Jobs* to *Slots* available in the Cluster. The *Scheduling* also acts as a Queue and submits Jobs into the slots in order once *Slots* become available.

The Batch Job CR is used to Model a reoccurring Batch Job Application. In order to support both Flink and Spark Applications, an abstract Batch Job CR is chosen that maps the state of application-specific CR (link SparkApplications[@SparkOperator] and FlinkCluster[@FlinkOperator]) to a common set of possible States. (Information about possible States and the corresponding State Machine are discussed in the Batch Job Operator Section). A Batch Job can be claimed by exactly one *Scheduling*, this is because the Batch Job CR models exactly the life cycle of a single application.

The Slots CR guarantees Resources in the Cluster by creating Ghost Pods, with a specific Resource Request representing the Size of an empty Slot. The Ghost Pods reserve resources by not allowing other Pods requiring scheduling to be scheduled onto the same Node. Using the extender and Preemption, the Slot Reconciler can reserve resources for Pods created by the Batch Job CR.

Finally, the Scheduling CR is passed to the external Interface by the External-Scheduler. Given a Set of BatchJobs and the Slots and the Node they exist on, an External-Scheduler can compute a *Scheduling*  that chooses BatchJobs and the Slots they should run in.

**Note**: Reconciler and Control-Loop can be used interchangeably, but for less confusion with the Spring Boot Concept of a Controller, the principle of a Control-Loop will be called Reconciler**
