# ingress-annotator
The `ingress-annotator` is a Kubernetes controller designed to manage and reconcile annotations on Ingress resources based on rules defined in a ConfigMap. It ensures that the specified annotations are correctly applied to Ingress resources and maintains the state of applied rules. The primary purpose of this controller is to automate the management of Ingress annotations, making it easier to apply and update configurations across multiple Ingress resources in a Kubernetes cluster.

## Description
The `ingress-annotator` project consists of two main reconcilers: `ConfigMapReconciler` and `IngressReconciler`.

### ConfigMapReconciler

The `ConfigMapReconciler` monitors a specific ConfigMap defined by its namespace and name. When changes are detected in this ConfigMap, it updates the internal data store (`RulesStore`) and triggers a reconciliation process for all Ingress resources in the cluster. During this process, it marks Ingress resources with a specific annotation (`annotatorReconcileNeededKey`) if they are enabled for reconciliation.

### IngressReconciler

The `IngressReconciler` handles the actual application of annotations to the Ingress resources. It fetches the current and last applied rules from the annotations, identifies any rules that need to be removed or updated, and applies the necessary changes. The reconciler then updates the Ingress resource's annotations to reflect the applied rules and the version of the ConfigMap from which the rules were sourced.

### Key Features

- **Automated Annotation Management**: Automatically applies and updates annotations on Ingress resources based on a ConfigMap.
- **Reconciliation Logic**: Ensures that only enabled Ingress resources are reconciled and that annotations are accurately managed.
- **Version Tracking**: Keeps track of the ConfigMap version to determine if an update is necessary for each Ingress resource.

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/ingress-annotatior:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/ingress-annotator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/ingress-annotator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/ingress-annotator/<tag or branch>/dist/install.yaml
```

## Contributing

We welcome contributions to the `ingress-annotator` project! Here are some ways you can help:

1. **Reporting Issues**: If you encounter any bugs or have suggestions for improvements, please open an issue on our GitHub repository.

2. **Submitting Pull Requests**: If you'd like to contribute code, please fork the repository and create a new branch for your changes. Ensure your code follows the project's coding standards and includes appropriate tests. Once your changes are ready, submit a pull request for review.

3. **Improving Documentation**: Good documentation is crucial for the success of any project. If you find gaps or inaccuracies in our documentation, feel free to submit updates.

4. **Feature Requests**: If you have ideas for new features or enhancements, we'd love to hear them! Open an issue to discuss your ideas with the community.

### Development Setup

1. **Clone the Repository**: Clone the repository to your local machine using `git clone`.
2. **Install Dependencies**: Ensure you have the necessary dependencies installed. This project typically requires Go and Kubernetes development tools.
3. **Build and Test**: Build the project using the provided Makefile or relevant build scripts, and run tests to verify your changes.

### Code of Conduct

We adhere to the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/0/code_of_conduct/). By participating in this project, you agree to abide by its terms.

Thank you for your interest in contributing to the `ingress-annotator` project!

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

