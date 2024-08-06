# ingress-annotator
`ingress-annotator` is a Kubernetes utility designed to dynamically manage ingress annotations based on predefined rules set in a ConfigMap. This tool simplifies the process of annotating ingresses in various namespaces, ensuring consistency and reducing manual configuration.

## Features

- **Dynamic Annotation Management**: Automatically applies annotations to ingresses based on the rules defined in a ConfigMap.
- **Namespace Specific Rules**: Apply annotations to ingresses in specified namespaces.
- **Ingress Specific Rules**: Apply annotations to specific ingresses within a namespace.
- **Wildcard Support**: Use wildcard patterns to match namespaces and ingress names.

## Usage
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ingress-annotator-rules
spec:
  # Rule to set proxy body size limit to 8MB for ingress resources 
  # in namespace prod1
  proxy-body-size: |
    annotations:
      ingress.kubernetes.io/proxy-body-size: "8m"
    namespace: "prod1"
    
  # Rule to rewrite the request URI to "/" for the specific ingress 
  # resource named ingress1 in namespace prod2
  rewrite-target: |
    annotations:
      ingress.kubernetes.io/rewrite-target: "/"
    namespace: "prod2"
    ingress: "ingress1"

  # Rule to configure OAuth2 authentication for ingress resources 
  # in namespaces dev1 and dev2
  oauth2-proxy: |
    annotations:
      nginx.ingress.kubernetes.io/auth-signin: "https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri"
      nginx.ingress.kubernetes.io/auth-url: "https://oauth2-proxy.example.com//oauth2/auth"
    namespace: "dev1,dev2"

  # Rule to set a whitelist of source IP ranges for ingress resources with 
  # names ending in "-priv" in namespaces that start with "dev"
  private: |
    annotations:
      nginx.ingress.kubernetes.io/whitelist-source-range: "192.168.1.0/24,10.0.0.0/16"
    namespace: "dev*"
    ingress: "*-priv"
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
