- Evaluate the Interface by using it to implement a simple scheduling algorithm
- Aim to outline features offered by the Interface: Such as multiple TestBeds + Saving Information on Batch Jobs
- Composed of a Profiler and the Scheduler that uses the Profiler Data
- Both Components run in Parallel.
- Profiler builds a Co-Location matrix. Scans for empty Entries in the Matrix
- Profiler Test Bed offers to empty slots on a single Node to force co-location.
- Profiler finds a missing co-location, both jobs are scheduled into the slots.
- Profiles measures time and stores the runtime in each batch-job. 

- If the scheduler gets asks to schedule a set of jobs, it will look up the information stored by the profiler, and greedily chooses co-locations.

