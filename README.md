# `auto-apps`

As of today, there is no formal support for the app of apps pattern, and the ArgoCD ecosystem lacks a construct where `Applications` can inherit easily from other `Applications`.

`auto-apps` is a stupid simple cli intended to bridge the gap until `ApplicationSets` are officially supported in ArgoCD. It is designed to fill a gap in the ArgoCD app of apps pattern, where multiple applications are bootstrapped from a single "umbrella" application.

## Features

### `Application` auto discovery

Starting from a given `spec.source.path`, `autoapps` will traverse a git repository looking for:

* valid `*.yaml` or `*.yml`
* manifests of `type: argoproj.io/v1alpha1` and `kind: Application`
* manifests with an annotation present of `autoapps`

### Simple `Application` templating

Using [custom argocd tooling](https://argoproj.github.io/argo-cd/user-guide/config-management-plugins/), `Applications` 
can utilize the `spec.source.plugin` spec to template child groups of `Applications`.

`autoapps` uses [fasttemplate](https://github.com/valyala/fasttemplate) to perform its substitution.  Since ArgoCD currently only supports specifying environment variables within plugins, `autoapps` accepts two forms of substitution:

__`AUTOAPPS_` prefixed__:

Variables prefixed with `AUTOAPPS_` are valid candidates for templating, and will have their prefix stripped before templating.  For example:

```yaml
# Parent Application
...
spec:
  source:
    plugin:
      name: autoapps
      env:
      - name: AUTOAPPS_foo
        value: bar
...

# Child Application
...
spec:
  source:
    path: {{foo}}
...
```

Identifying valid variables for templating via `AUTOAPPS_` prefix protects abuse against using protected or private system environment variables as template values.  However, one of the restrictions of this means you're limited to valid environment variable names.

__Builtin ArgoCD variables__:

ArgoCD uses a couple builtin environment variables at runtime that are useful:

```yaml
# Parent Application
...
spec:
  source:
    plugin:
      name: autoapps
...

# Child Application
...
spec:
  source:
    targetRevision: {{ARGOCD_APP_SOURCE_TARGET_REVISION}}
...
```

## Examples

Consider the following app of apps structure:

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
spec:
  project: {{ARGOCD_PROJECT}}

  source:
    repoURL: {{project_repo}}
    targetRevision: {{ARGOCD_APP_SOURCE_TARGET_REVISION}}    # leverage standard build environment variables
    path: apps/istio

  destination:
    server: https://kubernetes.default.svc
    namespace: istio-system
```

While the above is doable with any of the other templating tools, `autoapps` keeps the configuration entirely within
the `Application` CRD, enables runtime configuration, and simplifies "glue" templating with auto discovery of 
`Applications`.

### Dynamically skipping/ignoring applications

`autoapps` will skip any applications which are annotated as follows:

```yaml
# NOTE: Portions of the complete spec are skipped below for brevity
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: mocha
  annotations:
    autoapps-skip-discovery: "true"
...
```

To make this dynamic, you can leverage variable substitution from the parent application as follows:

```yaml
# Parent Application
...
spec:
  source:
    plugin:
      name: autoapps
      env:
      - name: AUTOAPPS_skip_mocha
        value: "true"
...

# Child Application
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: mocha
  annotations:
    autoapps-skip-discovery: "{{skip_mocha}}"
...
```
