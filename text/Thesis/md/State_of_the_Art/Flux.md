Flux aims to find a solution for Converged Computing, a term used, when describing the move of traditional HPC computing to the Cloud Native Computing Model. HPC Systems traditionally bring high performance and efficiency due to sophisticated scheduling, where the cloud offers resiliency, elasticity, portability and manageability. Traditional HPC batch scheduler, will not keep up with the growth of systems enabled by the cloud. 
Established HPC frameworks, such as Slurm [@wang2015towards], use a centralized scheduler. Flux Identifies scalability issues, that the scheduler is limited, in the maximum number of jobs it can handle. To prevent the scheduler from overwhelming, job submission needs to be throttled, which will decrease job throughput. While not strictly related to scheduling, a centralized approach, will also fail at tracking the status of jobs running inside larger clusters.

![Hierarchical Scheduling Approach (taken from Flux Poster [@FluxPoster])](graphics/flux_hierarchical_scheduler.pdf){#fig:fluxHierarchicalScheduler short-caption="Flux Hierarchical Scheduler"}

Flux introduces a new HPC scheduling model to address the challenges, by using one common resource and job management framework at both system and application levels. Using an hierarchical scheduler applying the divide-and-conquer approach to scheduling in a large cluster.
The hierarchical scheduling model, enables jobs to create their own schedulers, which is used to schedule its sub-jobs.

Another approach the Flux scheduler takes, to scale with the increasing number of jobs in a cluster, is to aggregate jobs, that are similar and arrive within the same time-frame, into single larger job.

Both Flux and the External-Scheduler-Interface could benefit from one another. Developing new scheduling algorithms integrated into the hierarchical scheduling approach could result in a scalable way of using many highly specific schedulers in a shared cluster.