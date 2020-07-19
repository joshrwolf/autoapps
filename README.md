# `auto-apps`

As of today, there is no formal support for the app of apps pattern, and the ArgoCD ecosystem lacks a construct where `Applications` can inherit easily from other `Applications`.

`auto-apps` is a stupid simple cli intended to bridge the gap until `ApplicationSets` are officially supported in ArgoCD.

`auto-apps` is designed to fill a gap in the ArgoCD app of apps pattern, where multiple applications are bootstrapped from a single "umbrella" application.

`auto-apps` solves this by the following:

__`Application` auto discovery__:

Starting from a given `spec.source.path`, `autoapps` will traverse a git repository looking for:

* valid `*.yaml` or `*.yml`
* manifests of `type: argoproj.io/v1alpha1` and `kind: Application`
* manifests with an annotation present of `autoapps`

__Isolated `Application` substitution__

Using [custom argocd tooling](https://argoproj.github.io/argo-cd/user-guide/config-management-plugins/), `Applications` 
can utilize the `spec.source.plugin` spec to template child groups of `Applications`.

```yaml
spec:
  source:
    plugin:
      name: autoapps
      env:
      - name: ARGOCD_SOME_VAR
        value: donkey
```

## Examples

Consider the following app of apps structure

```bash
# Application A syncs Application B and C, which subsequently deploy manifests
          +---+
          | B | --> App B Manifests
+---+     +---+
| A | --> 
+---+     +---+
          | C | --> App C Manifests
          +---+
```

If you want `B` and `C` to dynamically inherit values completely encapsulated within `A`'s manifests:

The "umbrella" application:

```yaml
# Application A CRD (umbrella app)
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: umbrella
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: core

  source:
    repoURL: https://github.com/joshrwolf/core-bootstrap
    targetRevision: HEAD
    path: "."   # Leverage `autoapps` traversal discovery

    plugin:
      env:
      - name: FOO
        value: bar

  destination:
    server: https://kubernetes.default.svc
    namespace: sample
```

The subsequent "child" application

```yaml
# Application B CRD (child app)
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: mocha
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
  annotations:
    argocd.argoproj.io/sync-wave: "1"
    autoapps: "true"
spec:
  project: ${ARGOCD_PROJECT}

  source:
    repoURL: https://github.com/joshrwolf/core-bootstrap
    targetRevision: ${ARGOCD_APP_SOURCE_TARGET_REVISION}    # leverage standard build environment variables
    path: apps/istio

  destination:
    server: https://kubernetes.default.svc
    namespace: istio-system
```

While the above is doable with any of the other templating tools, `autoapps` keeps the configuration entirely within
the `Application` CRD, enables runtime configuration, and simplifies "glue" templating with auto discovery of 
`Applications`.

## Work in progress

* "safe" environment variable substitution
