The functionality of the External-Scheduler-Interface is tested using a combination of Unit Tests and Integration Tests.

Unit tests are highly specific towards smaller system components and thus will not fit the written thesis's scope. They are located within the repository (Appendix).

Integration Tests aim to test the bigger picture. Testing all functionalities is not feasible in a system composed of many distributed processes, as setup would require recreating a cluster with all its software components. However, testing the complete system with all its components is possible with an already established cluster. The Manual scheduler aims to ease the use of External-Scheduler-Interface. 

For the integration tests to run in a timely manner, a common practice is to *mock* Software components, which are either not under immediate control or are very expensive to set up. The Java Operator SDK and the Fabric8 Kubernetes client can Mock the Kubernetes Cluster. Mocking the entirety of Kubernetes can not be accomplished, but the Kube-API-Server itself is enough to test the Operators. With the Kubernetes API-Server mocked, we can simulate the abstract state (declared state) and verify that changes to the abstract state trigger the correct actions by the Reconcilers.

Overview of Software Components used in the Integration Tests:

- Kubernetes API: For testing purposes, the Kubernetes API is just a CRUD REST application that can create, read, update, and delete resources. The Fabric8 Kubernetes Client offers a `KubernetesMockServer` with sufficient capabilities to test the Operators.

- Kubernetes Scheduler: Pods submitted to the Kubernetes API will be treated like regular resources and thus will not be scheduled onto Nodes. The Kubernetes API does not include any controllers for Kubernetes native resources. A simple implementation for a Kubernetes Scheduler has been implemented to test the functionalities of the Extender.

- Application Operator: Both of the application Operators are implemented in Go. Integrating them into the integration tests is out of the scope of this work, if not even unreasonable. Applications supposedly created by the application Operators are managed programmatically throughout the test cases.

- Batch Job, Scheduling, and Testbed Operator: The controller and the generated CRDs are used for the integration tests. Controllers are configured to use the `KubernetesMockServer`.

Basic functionality of the Reconciler components, at least on the resources manifests level, can be verified using the integration tests.

The Manual Scheduler Frontend can be used to test the External-Scheduler-Interface on an established Cluster with all components installed.

**Note:** At the time of writing the Fabric8 Kubernetes Server Mock does not appear to be fully thread safe which results in flaky tests. With the version 6.0.0 concurrency issues seem to be resolved.

## Manual Scheduler
The Manual Scheduler is a web frontend that interacts with the interface and allows a user to create a scheduling and test it. The Frontend shown in figure \ref{fig:manualScheduler} visualizes the current state of Testbeds, Schedulings, and Jobs. The scheduling builder allows to to quickly create a scheduling, by selecting a Testbed BatchJobs and their slots. To prevent unnecessary polling of the interface for changes, the frontend application connects to websockets and listens for changes.

![Manual Scheduler \label{fig:manualScheduler}](graphics/manual-scheduler.png)