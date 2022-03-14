The Operator Pattern is commonly used to extend Kubernetes Functionalities.

The Operator Pattern describes a Control-Loop that listens to the desired State of the Cluster and observes the actual state.
If the actual state diverges from the desired state, it does the necessary actions to move the actual state into the desired state.

The Operator pattern is based on the already existing design used by Kubernetes Native Resources like Deployments the *Control-Loop*.

Resources like Jobs, Deployments, and Statefulsets describe Pods' desired state. Essentially Kubernetes has controllers that monitor changes to the Resources Manifest and the current cluster situation. 
The Control-Loop allows a Deployment consisting of a Pod template with a Replication-Factor to the exact amounts of pods running inside the cluster. If any of the Pods fails, the Deployment Controller creates a new one. But on the Flip-Side, the controller also knows when the Resources Manifest is updated. For example, if the container is updated to a more recent version, all containers need to be replaced with the newer version. Deployment Resources have many policies that dictate how these actions should happen. Usually, restarting all Pods at once would not be desired so that the deployment would allow for Rolling-Upgrades. The controller ensures that new Pods are created first and become ready before old Pods are deleted.

Most of the time, Using just a new Controller is not enough to extend Kubernetes. It usually also requires Custom Resource Definitions.
CustomResourceDefinitions (CRD) is the Kubernetes way of defining which *kind* of resources are allowed to exist in the cluster. Having multiple Controllers listing to the same Resource, like a Deployment, makes little sense or could even cause issues. Thus the Combination of a new Resource and a Controller that knows how to handle it creates the Operator Pattern. The term *Operator*is used as the controller is created to replace previously manual work of configuring Kubernetes Native Resources done by an Operator. A common use case for the Operator Pattern is to Control Applications at a Higher level, where previously Multiple Deployments and Services may have been required to operate a Database. The Operator Pattern could reduce that to just a single Manifest containing the meaningful configuration. Operators can thus be created by Experts operating the Software and be used by any Kubernetes Cluster.

*(TODO: Best Practices when implementating an Operator)*

## Custom Resource Definitions

- Kubernetes resources are managed with a RESTful api, where Resources can be queried, created, updated and deleted
- CRD creates a new RESTful resource path which is managed by the Kubernetes API server. 
- Each version gets its on api
- Resources can be namespaced or cluster-scoped
- Custom resources, require a structural schema
    - Non Empty types: (properties for Objects), (items for arrays)
