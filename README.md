# ingress-annotator

[![release](https://github.com/kuoss/ingress-annotator/actions/workflows/release.yml/badge.svg)](https://github.com/kuoss/ingress-annotator/actions/workflows/release.yml)
[![pull-request](https://github.com/kuoss/ingress-annotator/actions/workflows/pull-request.yml/badge.svg)](https://github.com/kuoss/ingress-annotator/actions/workflows/pull-request.yml)
[![GitHub license](https://img.shields.io/github/license/kuoss/ingress-annotator.svg)](https://github.com/kuoss/ingress-annotator/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/kuoss/ingress-annotator)](https://goreportcard.com/report/github.com/kuoss/ingress-annotator)

The **Ingress Annotator** is a Kubernetes utility designed to streamline the management and application of annotations across Ingress resources and entire namespaces. With this tool, you can define reusable annotation rules in a ConfigMap, which are automatically propagated to your Ingresses or Namespaces based on simple annotation references. This ensures that updates to your annotation rules are immediately and consistently applied across your Kubernetes environment, reducing the risk of errors and making your deployments more maintainable.

## Features
- **Centralized Annotation Management**: Define reusable annotations in a ConfigMap that can be applied to multiple Ingress resources or entire namespaces. This ensures consistency and reduces the need for repetitive configurations.
- **Flexible and Scalable Application**: Apply annotation rules to individual Ingress resources or automatically propagate them to all Ingresses within a namespace, simplifying configuration management across your Kubernetes environment.
- **Dynamic and Automatic Updates**: Any changes to the annotation rules in the ConfigMap are automatically applied to all relevant Ingress resources or namespaces. Use the `annotator.ingress.kubernetes.io/rules` annotation on Ingresses or namespaces to dynamically control which rules are applied, ensuring precise and up-to-date configurations.

## Usage
1. Create a ConfigMap with your annotation rules:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ingress-annotator
  namespace: ingress-annotator
data:
  # Annotation rules that can be referenced in Ingress or Namespace annotations
  rules: |
    proxy-body-size:
      nginx.ingress.kubernetes.io/proxy-body-size: "8m"
    rewrite-target:
      nginx.ingress.kubernetes.io/rewrite-target: "/"
    oauth2-proxy:
      nginx.ingress.kubernetes.io/auth-signin: "https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri"
      nginx.ingress.kubernetes.io/auth-url: "https://oauth2-proxy.example.com/oauth2/auth"
    private:
      nginx.ingress.kubernetes.io/whitelist-source-range: "192.168.1.0/24,10.0.0.0/16"
```

2. Apply the ConfigMap:

```
kubectl apply -f configmap.yaml
```

3. Annotate Ingresses or Namespaces using the annotation `annotator.ingress.kubernetes.io/rules` as follows:

For Ingress:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress1
  namespace: namespace1
  annotations:
    annotator.ingress.kubernetes.io/rules: "oauth2-proxy,private"
    ...
```

For Namespace:
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: namespace1
  annotations:
    annotator.ingress.kubernetes.io/rules: "oauth2-proxy,private"
    ...
```

4. Verify that the annotations have been applied to the specified Ingress resources:
```
kubectl get ingress <ingress-name> -n <namespace> -o yaml
```

Example output:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress1
  namespace: namespace1
  annotations:
    annotator.ingress.kubernetes.io/managed-annotations: "{\"annotator.ingress.kubernetes.io/rules\":\"oauth2-proxy,private\",\"nginx.ingress.kubernetes.io/auth-signin\":\"https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri\",\"nginx.ingress.kubernetes.io/auth-url\":\"https://oauth2-proxy.example.com/oauth2/auth\",\"nginx.ingress.kubernetes.io/whitelist-source-range\":\"192.168.1.0/24,10.0.0.0/16\"}"
    annotator.ingress.kubernetes.io/rules: "oauth2-proxy,private"
    nginx.ingress.kubernetes.io/auth-signin: "https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri"
    nginx.ingress.kubernetes.io/auth-url: "https://oauth2-proxy.example.com/oauth2/auth"
    nginx.ingress.kubernetes.io/whitelist-source-range: "192.168.1.0/24,10.0.0.0/16"
    ...
```

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
