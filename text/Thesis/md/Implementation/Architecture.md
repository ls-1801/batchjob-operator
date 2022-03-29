In this section, all components that make up the interface are introduced. Here an architectural overview is presented, and interactions between components are examined.

The current implementation of the External-Scheduler-Interface consists of 5 components that will be introduced in this section but discussed in more detail in the Operator section.

![Components](graphics/architecture.pdf)

The five components consist of three Operators, each managing its resources. The architecture also uses an Extender and the external scheduler facing Web-API.

Additionally to the three reconcilers, three CRDs were designed.

- **BatchJob** represents an abstract batch application and can store information the external scheduler may want to remember for future invocations

* **Testbeds** represent a testbed of guaranteed resources available for a Scheduling. A Testbed CR is a collection of slots across the clusters Node that are referenced in a Scheduling

- **Schedulings** represents the decision done by the external scheduler. A Scheduling maps multiple BatchJobs to Testbeds available in the cluster. The scheduling also acts as a queue and submits jobs into the slots in order once slots become available.

The BatchJob CR is used to model a reoccurring batch application. To support both Flink and Spark applications, an abstract BatchJob CR is chosen that maps the state of application-specific CR (link SparkApplications[@SparkOperator] and FlinkCluster[@FlinkOperator]) to a common set of possible states. A BatchJob can be claimed by exactly one scheduling. This is because the BatchJob CR models exactly the life cycle of a single application.

Using the extender and preemption, the Testbed reconciler can reserve resources for Pods created by the BatchJob CR. The Slots CR guarantees resources in the cluster by creating Ghost Pods, with a specific resource request representing the size of a slot. The Ghost Pods reserve resources by preventing other Pods requiring scheduling to be scheduled onto the same Node.

Finally, the Scheduling CR is passed to the External-Scheduler-Interface by the external scheduler. Given a set of BatchJobs and the slots and the Node they exist on, an external scheduler can compute a Scheduling that chooses BatchJobs and the slots they should run in.