# etcd-release
---

This is a [BOSH](http://bosh.io) release for [etcd](https://github.com/coreos/etcd).

* [CI](https://mega.ci.cf-app.com/pipelines/etcd)
* [Roadmap](https://www.pivotaltracker.com/n/projects/1382120)

###Contents

1. [Deploying](#deploying)
2. [Running Tests](#running-tests)
3. [Advanced](#advanced)

## Deploying

In order to deploy etcd-release you must follow the standard steps for deploying software with BOSH.

We assume you have already deployed and targeted a BOSH director. For more instructions on how to do that please see the [BOSH documentation](http://bosh.io/docs).

###1. Uploading a stemcell
Find the "BOSH Lite Warden" stemcell you wish to use. [bosh.io](https://bosh.io/stemcells) provides a resource to find and download stemcells.  Then run `bosh upload stemcell STEMCELL_URL_OR_PATH_TO_DOWNLOADED_STEMCELL`.

###2. Creating a release
From within the etcd-release director run `bosh create release --force` to create a development release.

###3. Uploading a release
Once you've created a development release run `bosh upload release` to upload your development release to the director.

###4. Generating a deployment manifest
We provide a set of scripts and templates to generate a simple deployment manifest. You should use these as a starting point for creating your own manifest, but they should not be considered comprehensive or production-ready.

In order to automatically generate a manifest you must install [spiff](https://github.com/cloudfoundry-incubator/spiff).  Once installed, manifests can be generated using `./scripts/generate_etcd_deployment_manifest [STUB LIST]` with the provided stubs:

1. director_uuid_stub

	The director_uuid_stub provides the uuid for the currently targeted BOSH director.
	```yaml
	---
	director_uuid: DIRECTOR_UUID
	```
2. instance_count_stub

	The instance count stub provides the ability to overwrite the number of instances of etcd to deploy. The minimal deployment of etcd is shown below:
	```yaml
	---
	instance_count_overrides:
	  etcd_z1:
	    instances: 1
	  etcd_z2:
	    instances: 0
	```

	NOTE: at no time should you deploy only 2 instances of etcd.
3. persistent_disk_stub

	The persistent disk stub allows you to override the size of the persistent disk used in each instance of the etcd job. If you wish to use the default settings provide a stub with only an empty hash:
	```yaml
	---
	persistent_disk_overrides: {}
	```

	To override disk sizes the format is as follows
	```yaml
	---
	persistent_disk_overrides:
	  etcd_z1: 1234
	  etcd_z2: 1234
	```

4. iaas_settings

	The IaaS settings stub contains IaaS-specific settings, including networks, cloud properties, and compilation properties. Please see the BOSH documentation for setting up networks and subnets on your IaaS of choice. We currently allow for three network configurations on your IaaS: etcd1, etcd2, and compilation. You must also specify the stemcell to deploy against as well as the version (or latest).

We provide [default stubs for a BOSH-Lite deployment](https://github.com/cloudfoundry-incubator/etcd-release/blob/master/manifest-generation/bosh-lite-stubs).  Specifically:

* instance_count_stub: [manifest-generation/bosh-lite-stubs/instance-count-overrides.yml](manifest-generation/bosh-lite-stubs/instance-count-overrides.yml)
* persistent_disk_stub: [manifest-generation/bosh-lite-stubs/persistent-disk-overrides.yml](manifest-generation/bosh-lite-stubs/persistent-disk-overrides.yml)
* iaas_settings: [manifest-generation/bosh-lite-stubs/iaas-settings-etcd.yml](manifest-generation/bosh-lite-stubs/iaas-settings-etcd.yml)

```
mkdir -p tmp
cat >tmp/uuid.yml <<EOS
director_uuid: $(bosh status --uuid)
EOS
./scripts/generate_etcd_deployment_manifest tmp/uuid.yml \
	./manifest-generation/bosh-lite-stubs/instance-count-overrides.yml \
	./manifest-generation/bosh-lite-stubs/persistent-disk-overrides.yml \
	./manifest-generation/bosh-lite-stubs/iaas-settings-etcd.yml \
	./manifest-generation/bosh-lite-stubs/property-overrides.yml > tmp/manifest.yml
bosh deployment tmp/manifest.yml
```

[Optional]

1. If you wish to override the name of the release and the deployment (default: etcd) you can provide a release_name_stub with the following format:

	```yaml
	---
	name_overrides:
	  release_name: NAME
	  deployment_name: NAME
	```

Output the result of the above command to a file: `./scripts/generate_etcd_deployment_manifest [STUB LIST] > OUTPUT_MANIFEST_PATH`.

###5. Deploy

Run `bosh -d OUTPUT_MANIFEST_PATH deploy`.

## Running Tests

We have written a test suite that exercises spinning up single/multiple etcd instances, scaling them
and perform rolling deploys. If you have already installed Go, you can run `EATS_CONFIG=[config_file.json] ./scripts/test`.
The `test` script installs all dependancies and runs the full test suite. The EATS_CONFIG
environment variable points to a configuration file which specifies the endpoint of the BOSH
director and the path to your iaas_settings stub. An example config json for BOSH-lite would look like:

```json
cat > integration_config.json << EOF
{
  "bosh_target": "192.168.50.4",
  "iaas_settings_etcd_stub_path": "./src/acceptance-tests/manifest-generation/bosh-lite-stubs/iaas-settings-etcd.yml",
  "iaas_settings_turbulence_stub_path": "./src/acceptance-tests/manifest-generation/bosh-lite-stubs/iaas-settings-turbulence.yml",
  "turbulence_properties_stub_path": "./src/acceptance-tests/manifest-generation/bosh-lite-stubs/turbulence/property-overrides.yml",
  "cpi_release_url": "https://bosh.io/d/github.com/cppforlife/bosh-warden-cpi-release?v=21",
  "cpi_release_name": "bosh-warden-cpi",
  "turbulence_release_url": "http://bosh.io/d/github.com/cppforlife/turbulence-release?v=0.4"
}
EOF
export EATS_CONFIG=$PWD/integration_config.json
```

The full set of config parameters is explained below:
* `bosh_target` (required) Public BOSH IP address that will be used to host test environment.
* `iaas_settings_etcd_stub_path` (required) Stub containing iaas settings for the etcd deployment.
* `iaas_settings_turbulence_stub_path` (required for turbulence tests) Stub containing iaas setting for the turbulence deployment.
* `turbulence_properties_stub_path` (required for turbulence tests) Stub containing property overrides for the turbulence deployment.
* `cpi_release_url` (required for turbulence tests) CPI for the current BOSH director being used to deploy tests with.
* `cpi_release_name` (required for turbulence tests) Name for the `cpi_release_url` parameter
* `bosh_operation_timeout` (optional) Time to wait for BOSH commands to exit before erroring out. (default time is 5 min if not specified)
* `turbulence_operation_timeout` (optional) Time to wait for Turbulence operations to succeed before erroring out. (default time is 5 min if not specified)

Currently you cannot specify individual tests to be run.

Note: you must ensure that the stemcells specified in `iaas_settings_etcd_stub_path` and `iaas_settings_turbulence_stub_path` are already uploaded to the director at `bosh_target`.

Note: The ruby `bundler` gem is used to install the correct version of the `bosh_cli`, as well as to decrease the `bosh` startup time.

## Advanced


### Encrypting Traffic

To force communication between clients and etcd to use SSL, enable the etcd.require_ssl manifest property to true.

To force communication between etcd nodes to use SSL, set the `etcd.peer_require_ssl` manifest property to true.

The instructions below detail how to create certificates. If SSL is required for client communication, the clients will also need copies of the certificates.

When either SSL option is enabled, communication to the etcd nodes is done by consul DNS addresses rather than by IP address. When SSL is disabled, IP addresses are used and consul is not a dependency.

### Generating SSL Certificates

For generating SSL certificates, we recommend [certstrap](https://github.com/square/certstrap).
An operator can follow the following steps to successfully generate the required certificates.

> Most of these commands can be found in [scripts/generate-certs](scripts/generate-certs)

1. Get certstrap
   ```
   go get github.com/square/certstrap
   cd $GOPATH/src/github.com/square/certstrap
   ./build
   cd bin
   ```

2. Initialize a new certificate authority.
   ```
   $ ./certstrap init --common-name "etcdCA"
   Enter passphrase (empty for no passphrase): <hit enter for no password>

   Enter same passphrase again: <hit enter for no password>

   Created out/etcdCA.key
   Created out/etcdCA.crt
   ```

   The manifest property `properties.etcd.ca_cert` should be set to the certificate in `out/etcdCA.crt`

3. Create and sign a certificate for the etcd server.
   ```
   $ ./certstrap request-cert --common-name "etcd.service.consul" --domain "*.etcd.service.consul,etcd.service.consul"
   Enter passphrase (empty for no passphrase): <hit enter for no password>

   Enter same passphrase again: <hit enter for no password>

   Created out/etcd.service.consul.key
   Created out/etcd.service.consul.csr

   $ ./certstrap sign etcd.service.consul --CA etcdCA
   Created out/etcd.service.consul.crt from out/etcd.service.consul.csr signed by out/etcdCA.key
   ```

   The manifest property `properties.etcd.server_cert` should be set to the certificate in `out/etcd.service.consul.crt`
   The manifest property `properties.etcd.server_key` should be set to the certificate in `out/etcd.service.consul.key`

4. Create and sign a certificate for etcd clients.
   ```
   $ ./certstrap request-cert --common-name "clientName"
   Enter passphrase (empty for no passphrase): <hit enter for no password>

   Enter same passphrase again: <hit enter for no password>

   Created out/clientName.key
   Created out/clientName.csr

   $ ./certstrap sign clientName --CA etcdCA
   Created out/clientName.crt from out/clientName.csr signed by out/etcdCA.key
   ```

   The manifest property `properties.etcd.client_cert` should be set to the certificate in `out/clientName.crt`
   The manifest property `properties.etcd.client_key` should be set to the certificate in `out/clientName.key`

5. Initialize a new peer certificate authority. [optional]
   ```
   $ ./certstrap --depot-path peer init --common-name "peerCA"
   Enter passphrase (empty for no passphrase): <hit enter for no password>

   Enter same passphrase again: <hit enter for no password>

   Created peer/peerCA.key
   Created peer/peerCA.crt
   ```

   The manifest property `properties.etcd.peer_ca_cert` should be set to the certificate in `peer/peerCA.crt`

6. Create and sign a certificate for the etcd peers. [optional]
   ```
   $ ./certstrap --depot-path peer request-cert --common-name "etcd.service.consul" --domain "*.etcd.service.consul,etcd.service.consul"
   Enter passphrase (empty for no passphrase): <hit enter for no password>

   Enter same passphrase again: <hit enter for no password>

   Created peer/etcd.service.consul.key
   Created peer/etcd.service.consul.csr

   $ ./certstrap --depot-path peer sign etcd.service.consul --CA peerCA
   Created peer/etcd.service.consul.crt from peer/etcd.service.consul.csr signed by peer/peerCA.key
   ```

   The manifest property `properties.etcd.peer_cert` should be set to the certificate in `peer/etcd.service.consul.crt`
   The manifest property `properties.etcd.peer_key` should be set to the certificate in `peer/etcd.service.consul.key`

### Custom SSL Certificate Generation

If you already have a CA, or wish to use your own names for clients and
servers, please note that the common-names "etcdCA" and "clientName" are
placeholders and can be renamed provided that all clients client certificate.
The server certificate must have the common name `etcd.service.consul` and
must specify `etcd.service.consul` and `*.etcd.service.consul` as Subject
Alternative Names (SANs).
