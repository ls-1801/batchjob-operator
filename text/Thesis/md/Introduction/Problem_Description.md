Cluster resource managers, like YARN, emerging from Apache Hadoop, were centered around batch frameworks.

With the rise of cloud computing and all the benefits that come with it, companies were quick to adopt new cloud computing concepts. The concept of a cluster resource manager introduced a notion of simplicity to those developing applications for the cloud. A cluster resource manager now manages many aspects that used to be handled by dedicated operations teams.

Kubernetes, a cluster resource manager that was initially developed by Google after years of internal use, provided an all-around approach to cluster resource management for not just batch application. The global adoption of Kubernetes by many leading companies led to the growth of the ecosystem around it. Kubernetes has grown a lot since and has become the new industry standard, benefiting from a vast community. 

Old batch-application-focused cluster resource managers that used to be the industry standard are being pushed away by Kubernetes. Unfortunately, vastly different interfaces or scheduling mechanism between other cluster resource managers usually block the continuation of existing research done in batch scheduling algorithms.

Finding an efficient scheduling algorithm is a complex topic in itself. Usually, the setup required to further research existing scheduling algorithms is substantial. Dealing with different cluster resource manager further complicates continuing on already existing work.