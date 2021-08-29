# Bytebase API Style Guide

Bytebase uses REST API and this doc describes the corresponding API style guide.

The guiding prinicipal for our style guide is **consistency**.

# Methods

## Prefer PATCH over PUT

Most of the time, we only want to do partial update on the resource, and we should use PATCH accordingly. PUT on the other hand means to overwrite the entire resource with the request fields and would more likely to reset existing fields unexpectedly.

# Resource URL naming

## Use resource id for addressing the specific resource

Bytebase uses auto incremental ID as the primary key for all resources. To address a particular resource, we use GET `/issue/42`, if we want to support other addressing mechanism like using resource name, we should use query parameter like `.issue/42?name=foo`

## Use lower case and do not use separator to split the words

Use `/foo/barbaz` instead of `/foo/barBaz` or `/foo/bar-baz`

_Rationale_: The rule is simple and thus improve consistency. e.g. Say we have a resource called `datasource`, both `data source` and `datasource` are acceptable terms, under our rule, it's always `datasource`. This does hurt readability sometimes, but most of the time, we should only have at most 2 words inside a path component of the URL and our brain is pretty good at recognizing the individual word.

_Note_: It's more common to use camelCase or dash-case. However, we are not alone, [Kubernetes](https://kubernetes.io/docs/reference/) also adopts this convention.

## Use singular form even for collection resource

Use `GET /issue` instead of `GET /issues` to fetch the list of issues.

_Rationale_: Plural forms have several variations and it's hard for non-native English speakers to remember all the rules. And in practice, using singular form for collection resource won't cause confusion with the singular resource because they use different resource paths, e.g. `/issue` versus `/issue/:id`.

_Note_: We do aware this is different from the common convention. However, we are not alone, see [this Kubernetes discussion](https://github.com/kubernetes/kubernetes/issues/18622).

## Use a separate `/{{resource}}/batch` for batch operation

If the resource supports batch operation, then use a separate `/batch` endpoint under that resource.

# References

1. [Google's AIP](https://google.aip.dev/)
1. [Kubernetes API reference](https://kubernetes.io/docs/reference/)