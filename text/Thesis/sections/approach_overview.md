
During the initial phase of collecting Resources regarding influencing the Kubernetes Scheduler, I discovered multiple ways I could accomplish my goals.
This section shall give a brief overview of the different Approaches i could have chosen, and maybe provide an answer on why i did not chose them in the end.

   - Implementation of Scheduling Algorithm, like Hugo and Mary dirctly inside the Kube-Scheduler, either by replacing the original or extending with a plugin
   - Implementation of the Interface directly inside the Kube Scheduler, forgoing the Operator/Controller
   - Implementation of the Interface as an Extender