# etcd-release
---

###About

This is a [bosh](http://bosh.io) release for [etcd](https://github.com/coreos/etcd).

###Usage

The `etcd.machines` property must be specified in your manifest. It should be an array of the form `[<ip>:<port>, ...]`.

###Etc

* Currently this release is not consumed by Cloud Foundry as a stand-alone bosh release. This is expected to change in the near future
* At no point should you deploy exactly two (2) instances of etcd. Please see [CoreOS's recommendations](https://coreos.com/docs/cluster-management/scaling/etcd-optimal-cluster-size/) for cluster sizes.
