# v1.2.6
2022-12-29
## Bugfix:
- fix choudshell using vim when type ↑↓←→ 
## Dependency:
- kubecube 1.4.0

# v1.2.4
2022-8-31
## Bugfix:
- add cluster info when request to kubecube for pod authorization
## Dependency:
- kubecube 1.4.0

# v1.2.3
2022-7-19
## Enhance:
- expose log config

# v1.2.2
2022-6-22
## Bugfix:
- fix leader election logic
## Enhance:
- add .gitignore file

# v1.2.1
2022-6-17
## Enhance:
- cloudshell shell: fix auth for kubecube
## Dependency:
- kubecube 1.2.1

# v1.2.0
2022-4-26
## Enhance:
- replace kubecube version v1.1.0 to v1.2.0
- replace jwt version v3.2.0 to v3.2.1
## Bugfix:
- fix webcosole leader selection logic (expose healthz api)
## Dependency:
- kubecube 1.2.0

# v1.1.0
2021-12-16
## Feature
- add audit feature
## Bugfix:
- fix webconsole auth request content-type

# v1.0.1
2021-9-3
## Bugfix:
- fix cloudshell shell: decode kubeconfig after wget

# v1.0.0
2021-8-6
## Feature
- Update version of dependent package kubecube from v1.0.0-rc to v1.0.0

# V1.0.0-rc0
2021-7-16
## Feature
- Webconsole: enter container in pods and post request
- Cloudshell: enter the specified container and call the script to isolate the user operation