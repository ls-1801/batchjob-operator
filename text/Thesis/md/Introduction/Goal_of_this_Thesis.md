To aid further research on Batch-Scheduling-Algorithms, the goal of this thesis is to provide a simplistic interface for batch scheduling on Kubernetes.

Already existing scheduling algorithms, like Mary and Hugo [@thamsen2021mary], were initially developed for the cluster resource manager YARN. Reusing existing scheduling algorithms on the nowadays broadly adopted cluster resource manager Kubernetes is not a trivial task due to vastly different interfaces and interactions with the cluster manager.

Extending existing research to the more popular resource manager Kubernetes provides multiple benefits.

1. The large ecosystem around Kubernetes allows for a better development environment due to debugging and diagnostic tooling
2. The time it took to set up a new development environment, including a cluster, has been significantly shortened. This is also due to the larger ecosystem surrounding Kubernetes. Tools like Minikube [@minikubegithub] provide a quick and easy way to set up a cluster on a local machine. Most cloud providers offer a managed version of Kubernetes [@8968907; @khallouli2021cluster].

The interface should provide easy access to the Kubernetes cluster, allowing an external scheduler to queue batch applications and move them into predefined slots inside the cluster.

For an external scheduler to form a scheduling decision, the interface should provide an overview of the current cluster situation containing:

1. Information about empty or in use slots in the cluster
2. Information about jobs in the Queue
3. Information about the history of reoccurring jobs, like the start and stop timestamp, that can be used to query metrics from a metrics provider.

It should be possible for an external scheduler to form a scheduling decision based on a collection of jobs and metrics collected from the cluster. The interface should accept the scheduling decision and translate it into Kubernetes concepts to establish the desired scheduling in the cluster.

Currently, the Kubernetes cluster resource manager does not offer the concept of a queue. Submitting jobs to the cluster would either allocate resources immediately or produce an error due to missing resources.

Kubernetes provides various mechanisms to influence the placement of specific applications on specific Nodes. However, these might become unreliable on a busy cluster and require a deep understanding of Kubernetes concepts, thus creating a barrier for future research. Kubernetes does not offer the concept of dedicated slots for applications either.