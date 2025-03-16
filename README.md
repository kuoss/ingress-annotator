# ingress-annotator

[![release](https://github.com/kuoss/ingress-annotator/actions/workflows/release.yml/badge.svg)](https://github.com/kuoss/ingress-annotator/actions/workflows/release.yml)
[![pull-request](https://github.com/kuoss/ingress-annotator/actions/workflows/pull-request.yml/badge.svg)](https://github.com/kuoss/ingress-annotator/actions/workflows/pull-request.yml)
[![GitHub license](https://img.shields.io/github/license/kuoss/ingress-annotator.svg)](https://github.com/kuoss/ingress-annotator/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/kuoss/ingress-annotator)](https://goreportcard.com/report/github.com/kuoss/ingress-annotator)

The **Ingress Annotator** is a Kubernetes utility designed to streamline the management and application of annotations across Ingress resources within your clusters. With this tool, you can define reusable annotation rules in a ConfigMap, which are automatically propagated to your Ingresses based on simple selectors. This ensures that updates to your annotation rules are applied immediately and consistently across relevant Ingress resources, reducing the risk of errors and enhancing the maintainability of your deployments.

## Features
- **Centralized Annotation Management**: Define reusable annotations in a ConfigMap that can be applied to multiple Ingress resources. This promotes consistency and reduces the need for repetitive configurations.
- **Flexible and Scalable Application**: Apply annotation rules to individual Ingress resources automatically using powerful selection mechanisms, simplifying configuration management in your Kubernetes environment.
- **Dynamic and Automatic Updates**: Any changes to the annotation rules in the ConfigMap are automatically applied to all relevant Ingress resources. The ingress-annotator controller continuously reconciles Ingress annotations to ensure they remain up to date.
- 
## Usage
1. Create a ConfigMap with Annotation Rules

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ingress-annotator
  namespace: ingress-annotator
data:
  rules: |
    - description: ns1-auth
      selector:
        include: "ns1"  # Targets all Ingresses within the 'ns1' namespace.
      annotations:
        nginx.ingress.kubernetes.io/auth-signin: "https://oauth.example.com/oauth2/start?rd=https://$host$request_uri"
        nginx.ingress.kubernetes.io/auth-url: "https://oauth.example.com/oauth2/auth"

    - description: restrict-access-except-public
      selector:
        include: "*"  # Includes all Ingresses (wildcard matches every namespace and name).
        exclude: "frontend/landing-page,dashboard/public-status" # Excludes specific Ingresses.
      annotations:
        nginx.ingress.kubernetes.io/whitelist-source-range: "10.0.0.0/16"

    - description: api-secure-mtls
      selector:
        include: "api/*"  # Targets all Ingresses within the 'api' namespace.
        exclude: "api/internal,api/debug"  # Excludes specific Ingresses.
      annotations:
        nginx.ingress.kubernetes.io/auth-tls-secret: "api/mtls-secret"

    - description: throttle
      selector:
        include: "*/throttle-*,*/*-throttle"  # Targets specific Ingress patterns.
      annotations:
        nginx.ingress.kubernetes.io/limit-rate: "10"
```

2. Apply the ConfigMap

```
kubectl apply -f configmap.yaml
```

3. Verify that Annotations are Applied

```
kubectl get ingress <ingress-name> -n <namespace> -o yaml
```

Example output:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress1
  namespace: ns1
  annotations:
    ingress-annotator.kuoss.io/managed-annotations: >
      {"nginx.ingress.kubernetes.io/auth-signin":"https://oauth.example.com/oauth2/start?rd=https://$host$request_uri",
       "nginx.ingress.kubernetes.io/auth-url":"https://oauth.example.com/oauth2/auth"}
    nginx.ingress.kubernetes.io/auth-signin: "https://oauth.example.com/oauth2/start?rd=https://$host$request_uri"
    nginx.ingress.kubernetes.io/auth-url: "https://oauth.example.com/oauth2/auth"
    ...
```

## Notes
- Ingress resources are updated based on the defined rules in the ConfigMap.
- Ingress resources are dynamically updated when the ConfigMap changes, ensuring real-time updates across clusters.
- Annotations managed by `ingress-annotator` are stored under `ingress-annotator.kuoss.io/managed-annotations` to prevent conflicts with manually added annotations.
