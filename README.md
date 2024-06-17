# Reforma

![GitHub Release](https://img.shields.io/github/v/release/prosimcorp/reforma)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/prosimcorp/reforma)
[![Go Report Card](https://goreportcard.com/badge/github.com/prosimcorp/reforma)](https://goreportcard.com/report/github.com/prosimcorp/reforma)
![image pulls](https://img.shields.io/badge/+2k-brightgreen?label=image%20pulls)
![GitHub License](https://img.shields.io/github/license/prosimcorp/reforma)

![GitHub User's stars](https://img.shields.io/github/stars/prosimcorp?label=Prosimcorp%20Stars)
![GitHub followers](https://img.shields.io/github/followers/prosimcorp?label=Prosimcorp%20Followers)

> **ATTENTION:** From v0.4.0+ bundled Kubernetes deployment manifests are built and uploaded to the releases. 
> We do this to keep them atomic between versions. Due to this, `deploy` directory will be removed from repository. 
> Please, read [related section](#deployment)

## Description
Kubernetes operator to patch resources with information from other resources

## Motivation

The GitOps approach has demonstrated being the best way to keep the traceability and reproducibility of a deployment
for any project. Not only for developers' applications but for the SRE tools inside the cluster too. As always, challenges
have appeared around that way of doing things.

1. Several companies which works with Kubernetes, create a repository with the manifests of the tools they
   deploy on the cluster's creation (most times this is known as `Tooling Stack`). This is a good and simple approach, 
   but **working with several cloud providers at the same time** means that several distributions of this stack 
   must be maintained, most times, for the same exact stack just to change some little configurations, such as the flags.
   We preferred another path where the stack can discover information from inside Kubernetes, such as ConfigMap resources, 
   and modify itself dynamically to work.
   This way, people involved on maintenance only have to maintain one repository, that can be deployed in several cloud 
   providers at the same time, being able to automate the deployment using FluxCD or ArgoCD.


2. Sometimes, **ServiceAccount resources need annotations** to be able to modify cloud resources, such as DNS registries in 
   AWS Route53 for ExternalDNS or Cert manager (if you use ACME's DNS solver). Why not crafting this kind of annotations
   dynamically getting information from a ConfigMap? With this approach, your Terraform code can create IAM roles 
   following a pattern and ServiceAccounts can be automatically annotated.

## Deployment

We have designed the deployment of this project to allow remote deployment using Kustomize. This way it is possible
to use it with a GitOps approach, using tools such as ArgoCD or FluxCD. Just make a Kustomization manifest referencing
the tag of the version you want to deploy as follows:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- https://github.com/prosimcorp/reforma/releases/download/v0.4.0/bundle.yaml
```

> 🧚🏼 **Hey, listen! If you prefer to deploy using Helm, go to the [Helm registry](https://github.com/prosimcorp/helm-charts)**

## RBAC

We designed the operator to be able to patch any kind of resource in a Kubernetes cluster, but by design, Kubernetes
permissions are always only additive. This means that we had to grant only some resources to be patched by default,
such as Secrets and ConfigMaps. But you can patch other kind of resources just granting some permissions to the
ServiceAccount of the controller as follows:

```yaml
# clusterRole-reforma-custom-resources.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
   name: reforma-custom-resources
rules:
   - apiGroups:
        - "*"
     resources:
        - "*"
     verbs:
        - create
        - delete
        - get
        - list
        - patch
        - update
        - watch
---
# clusterRoleBinding-reforma-custom-resources.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
   name: reforma-custom-resources
roleRef:
   apiGroup: rbac.authorization.k8s.io
   kind: ClusterRole
   name: reforma-custom-resources
subjects:
   - kind: ServiceAccount
     name: reforma-controller-manager
     namespace: default
---
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: reforma

resources:
   - https://github.com/prosimcorp/reforma/releases/download/v0.4.0/bundle.yaml
   
   # Add your custom resources
   - clusterRole-reforma-custom-resources.yaml
   - clusterRoleBinding-reforma-custom-resources.yaml
```

## Example

To patch resources using this operator you will need to create a CR of kind Patch. You can find the spec samples
for all the versions of the resource in the [examples directory](./config/samples)

You may prefer to learn directly from an example, so let's explain it patching a ServiceAccount using information from a 
ConfigMap resource:

```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
      name: cluster-info
      namespace: kube-system
   data:
     account: "111111111111"
     environment: develop
     name: your-project-emea
     provider: AWS
     region: eu-west-1
```

Now use a Patch CR to patch the ServiceAccount:

```yaml
apiVersion: reforma.prosimcorp.com/v1beta1
kind: Patch
metadata:
   name: patch-external-dns-sa
spec:
   # Synchronization parameters
   synchronization:
      time: "5s"

   # Sources to look for the data to make wonderful patches
   sources:
      - apiVersion: v1
        kind: ConfigMap
        name: cluster-info
        namespace: kube-system

   # Target to apply patches to
   target:
      apiVersion: v1
      kind: ServiceAccount
      name: external-dns
      namespace: external-dns

   # You know, the patch type
   patchType: application/merge-patch+json

   # Templating section is where you can be creative to craft a patch
   # Basically, if you know Helm templating and Kustomize patches, do what you want
   template: |
     {{- $source := (index . 1) -}}
     metadata:
       annotations:          
         {{- if eq ($source.data.provider | lower) "aws" }}
         eks.amazonaws.com/role-arn: "arn:aws:iam::{{- $source.data.account -}}:role/{{- $source.data.name -}}-external-dns"
         {{- end }}

         {{- if eq ($source.data.provider | lower) "gcp" }}
         iam.gke.io/gcp-service-account: "{{- $source.data.name -}}-external-dns@{{- $source.data.account -}}.iam.gserviceaccount.com"
         {{ end }}
```

## Templating engine

### What you can use
Even when we recommend keeping the scope of the patches as small as possible, we wanted a powerful engine to do them, so 
we mixed several gears, from here and there, and got all the power of a wonderful toy.

In the end of this madness you are reading about, what you will notice is that you can basically use everything you 
already know from [Helm Template](https://helm.sh/docs/chart_template_guide/functions_and_pipelines/)

### How to use collected data
All the sources and the target are stored (and given) as a list of items, starting from the target (it is only one) followed 
by the sources (they can be many). This list of objects is available inside the template, into the main scope `.`

This means that the objects can be accessed or stored in variables in the following way:
```yaml
apiVersion: reforma.prosimcorp.com/v1beta1
kind: Patch
metadata:
  name: accessing-objects-sample
spec:
  .
  .
  .
  patchType: application/json-patch+json
  template: |
    {{- $target := (index . 0) -}}
    {{- $source := (index . 1) -}}
    {{- $another_source := (index . 2) -}}

    - op: add
      path: /metadata/annotations/cluster-name
      value: "{{- $source.metadata.name -}}"
```

## How to develop

> We recommend you to use a development tool like [Kind](https://kind.sigs.k8s.io/) or [Minikube](https://minikube.sigs.k8s.io/docs/start/)
> to launch a lightweight Kubernetes on your local machine for development purposes

For learning purposes, we will suppose you are going to use Kind. So the first step is to create a Kubernetes cluster
on your local machine executing the following command:

```console
kind create cluster
```

Once you have launched a safe play place, execute the following command. It will install the custom resource definitions
(CRDs) in the cluster configured in your ~/.kube/config file and run the Operator locally against the cluster:

```console
make install run
```

> Remember that your `kubectl` is pointing to your Kind cluster. However, you should always review the context your
> kubectl CLI is pointing to

## How releases are created

Each release of this operator is done following several steps carefully in order not to break the things for anyone.
Reliability is important to us, so we automated all the process of launching a release. For a better understanding of
the process, the steps are described in the following recipe:

1. Test the changes on the code:

    ```console
    make test
    ```

   > A release is not done if this stage fails


2. Define the package information

    ```console
    export VERSION="0.0.1"
    export IMG="ghcr.io/prosimcorp/reforma:v$VERSION"
    ```

3. Generate and push the Docker image (published on Docker Hub).

    ```console
    make docker-build docker-push
    ```

4. Generate the manifests for deployments using Kustomize

   ```console
    make bundle-build
    ```

## How to collaborate

This project is done on top of [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder), so read about that project 
before collaborating. Of course, we are open to external collaborations for this project. For doing it you must fork the 
repository, make your changes to the code and open a PR. The code will be reviewed and tested (always)

> We are developers and hate bad code. For that reason we ask you the highest quality on each line of code to improve
> this project on each iteration.

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Special mention

This project was done using IDEs from JetBrains. They helped us to develop faster, so we recommend them a lot! 🤓

<img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.png" alt="JetBrains Logo (Main) logo." width="150">