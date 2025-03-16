# ingress-annotator

[![release](https://github.com/kuoss/ingress-annotator/actions/workflows/release.yml/badge.svg)](https://github.com/kuoss/ingress-annotator/actions/workflows/release.yml)
[![pull-request](https://github.com/kuoss/ingress-annotator/actions/workflows/pull-request.yml/badge.svg)](https://github.com/kuoss/ingress-annotator/actions/workflows/pull-request.yml)
[![GitHub license](https://img.shields.io/github/license/kuoss/ingress-annotator.svg)](https://github.com/kuoss/ingress-annotator/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/kuoss/ingress-annotator)](https://goreportcard.com/report/github.com/kuoss/ingress-annotator)

The **Ingress Annotator** is a Kubernetes utility that automates the management of annotations across Ingress resources. By leveraging a ConfigMap-driven approach, it ensures consistent, dynamic, and conflict-free annotation application.

## Features
- **Centralized Annotation Management**: Define reusable annotation rules in a ConfigMap and apply them dynamically across multiple Ingress resources.
- **Flexible Rule-Based Selection**: Target specific namespaces or Ingress resources using selectors for precise control.
- **Automatic Updates**: Watches for ConfigMap changes and immediately applies updates to relevant Ingress resources.
- **Conflict Prevention**: Managed annotations are stored under `ingress-annotator.kubernetes.io/managed-annotations` to prevent conflicts with manually added annotations. If an annotation exists in both the ConfigMap and the Ingress resource, the ConfigMap value **overwrites** the existing annotation. However, manually added annotations that are not managed by `ingress-annotator` remain unchanged.


## Installation & Configuration

To install and configure Ingress Annotator, follow these steps:

1. **Create the Namespace**

   ```sh
   kubectl create namespace ingress-annotator
   ```

2. **Create a ConfigMap**

   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: ingress-annotator
     namespace: ingress-annotator
   data:
     rules: |
       - description: test
         selector:
           include: ingress-annotator
         annotations:
           hello: world
   ```

   Apply the ConfigMap:

   ```sh
   kubectl apply -f configmap.yaml
   ```

3. **Deploy Ingress Annotator**

   ```sh
   kubectl apply -f https://raw.githubusercontent.com/kuoss/ingress-annotator/refs/heads/main/deploy/ingress-annotator.yaml
   ```

   Verify that the deployment is running:

   ```sh
   kubectl -n ingress-annotator get pods
   ```

4. **Create an Ingress Resource for Testing**

   ```sh
   cat <<EOF | kubectl apply -f -
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
     name: test-ingress
     namespace: ingress-annotator
   spec:
     rules:
       - host: example.com
         http:
           paths:
             - path: /
               pathType: Prefix
               backend:
                 service:
                   name: test-service
                   port:
                     number: 80
   EOF
   ```

5. **Verify Annotations**

   Check if the annotations have been applied:

   ```sh
   kubectl get ingress test-ingress -n ingress-annotator -o yaml
   ```

   Example Output:

   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
     name: test-ingress
     namespace: ingress-annotator
     annotations:
       ingress-annotator.kubernetes.io/managed-annotations: >
         {"hello":"world"}
       hello: world
   ```

   If the annotation is missing, check the Ingress Annotator logs:

   ```sh
   kubectl logs -n ingress-annotator deploy/ingress-annotator
   ```

6. **Modify and Apply ConfigMap**

   After installation, you can modify the ConfigMap as needed to apply custom rules.

   To edit the existing ConfigMap:

   ```sh
   kubectl edit configmap ingress-annotator -n ingress-annotator
   ```

## Rule Examples

### IP Whitelisting for a Namespace

```yaml
- description: allow-specific-ips
  selector:
    include: "dev"
  annotations:
    nginx.ingress.kubernetes.io/whitelist-source-range: "68.204.79.0/27,68.204.135.80/28,246.91.35.0/24"
```

Alternatively, using a list format:

> [!NOTE]
> `listAnnotations` allows defining annotations in a structured list format, making it easier to manage multiple values and add descriptive comments.

```yaml
- description: allow-specific-ips
  selector:
    include: "dev"
  listAnnotations: # Ensure listAnnotations not annotations
    nginx.ingress.kubernetes.io/whitelist-source-range:
      - 68.204.79.0/27   # Dev Site 1
      - 68.204.135.80/28 # Dev Site 2
      - 246.91.35.0/24   # Testers
```

### Authentication (OAuth2)
```yaml
- description: ns1-oauth
  selector:
    include: "ns1"
  annotations:
    nginx.ingress.kubernetes.io/auth-signin: "https://oauth.example.com/oauth2/start?rd=https://$host$request_uri"
    nginx.ingress.kubernetes.io/auth-url: "https://oauth.example.com/oauth2/auth"
```

### Basic Authentication for Internal Services
```yaml
- description: enforce-basic-auth
  selector:
    include: "internal/*"
  annotations:
    nginx.ingress.kubernetes.io/auth-type: "basic"
    nginx.ingress.kubernetes.io/auth-secret: "internal/basic-auth"
```

### Rate Limiting for High-Traffic Endpoints
```yaml
- description: rate-limit-api
  selector:
    include: "api/throttle-*"
  annotations:
    nginx.ingress.kubernetes.io/limit-rate: "10" # Limits requests to 10 per second
```

### Enforcing HTTPS Redirects
```yaml
- description: enforce-https
  selector:
    include: "*"
  annotations:
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true" # Forces HTTP to HTTPS redirect
    nginx.ingress.kubernetes.io/ssl-redirect: "true" # Ensures HTTPS enforcement
```


## Selector Format

Selectors allow flexible matching using `include` and `exclude` fields, following the format: `namespace/ingress-name`.
These fields support:
- Multiple values as a comma-separated string (`include: "dev,prod"`).
- Wildcards (`*`) for broader matches (`include: "*"` applies to all namespaces and Ingress resources).
- Explicit namespace-only selection (`include: "dev"` selects all Ingress resources in the `dev` namespace).
- Explicit Ingress name selection (`include: "prod/hello-world"` selects only `hello-world` in the `prod` namespace).

### Include Rules
```yaml
include: "*"  # Applies to all Ingress resources
include: "dev"  # Applies to all Ingress resources in the 'dev' namespace
include: "prod/*"  # Applies to all Ingress resources in the 'prod' namespace
include: "dev,prod"  # Applies to both 'dev' and 'prod' namespaces
include: "prod/hello-*"  # Matches Ingress resources in 'prod' starting with 'hello-'
```

### Exclude Rules
```yaml
exclude: "prod/ingress1"  # Excludes 'ingress1' in 'prod'
exclude: "prod-*"  # Excludes all namespaces starting with 'prod-'
```
