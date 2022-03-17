To test the functionalities of the External-Scheduler-Interface, a combination of Unit Tests and Integration Tests are used.

Unit tests are highly specific towards smaller components of the System and thus will not fit in the scope of the written thesis, however they can be found inside the repository (Appendix).

Integration Tests aim to test the bigger picture. In a system composed of many distributed processes, testing all functionalities is not feasible, as setup would require recreating a cluster of software components. Testing the complete system with all its components, is only possible interacting with an established cluster. The Manual scheduler aims to ease the use of External-Scheduler-Interface. 

For Integration Test to run in a timely manner, a common practice is to *Mock* Software components, which are either not under immediate control or are very expensive to setup. The Java Operator SDK and the Fabric8 Kubernetes client, can Mock the Kubernetes Cluster. Mocking the entirety of Kubernetes can not be accomplished, but the Kube-API-Server itself is enough to test the Operators. With the Kubernetes API-Server mocked, we can simulate the abstract state (declared state) and verify that changes to the abstract state trigger the correct actions by the Reconcilers.

Overview of Software Components used in the Integration Tests:
- Kubernetes API: Mocked by Fabric8
- Batch Job Operator: Actual Reconciler + CRDs are used
- Application Operator (Spark): Mocking changes to the SparkApplication
- Scheduling Operator: Actual Reconciler + CRDs are used
- (Scheduler): Out of Scope for this work
- TestBed Operator: Actual Reconciler + CRDs are used. The actual Slot Occupation Status is mocked

Using integration tests, basic functionality of the Reconciler components, at least on the Abstract Resource level can be verified.

To test the External-Scheduler-Interface on an established Cluster with all components installed, the Manual Scheduler Frontend can be used.

## Manual Scheduler
The Manual Scheduler is a web frontend that interacts with the Interface, and allows a user of the Interface to create a scheduling and test it.

The fronted both acts as a reference on how the Scheduler-Interface is supposed to be used, and to verify the functionality of the Cluster. 

