# `auto-apps`

`auto-apps` is a simple cli intended to bridge the gap until `ApplicationSets` are officially supported in ArgoCD.

`auto-apps` is designed to fill a gap in the ArgoCD app of apps pattern, where multiple applications are bootstrapped from a single "umbrella" application.

As of today, there is no formal support for the app of apps pattern, and the ArgoCD ecosystem lacks a construct where `Applications` can inherit easily from other `Applications`.

`auto-apps` solves this by the following:

* Simple auto discovery of `Applications` that exist within a git repository
* Simple environment based substitution of helm-like defined templating

## Examples

Work in progress