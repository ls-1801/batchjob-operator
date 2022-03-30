
1. [https://github.com/ls-1801/scheduler-interface](https://github.com/ls-1801/scheduler-interface)

2. [https://github.com/ls-1801/ManualScheduler](https://github.com/ls-1801/ManualScheduler)

3. 

The source code for the External-Scheduler-Interface is hosted on github.com. Both repositories contain READMEs with details on installing and using individual components. The scheduler-interface repository also includes the example scheduler used in the evaluation.

The installation of the interface was tested on a new cluster in the "test-namespace" namespace. The namespace is based on the namespace of the Helm release and may or may not need to be created. The `Values.yaml` need to be adjusted when installing to a different Namespace. Images are publicly available in a repository. Thus, a complete Interface build is not required; however, build instructions are also included within the repository.

The Manual Scheduler is an Angular Web application and can either be built using the Angular CLI or a docker container, building and serving the application.
