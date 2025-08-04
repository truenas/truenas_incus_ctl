# truenas_incus_ctl

`truenas_incus_ctl` is a tool for administering datasets, snapshots and network shares that are hosted on a TrueNAS server.

## Install

`go build`

`go install`

## Login to a TrueNAS host to generate and store an API key

`truenas_incus_ctl config login` then follow the prompts to login to a TrueNAS host, and record the config into a config file. It is preferred to generate an API key.

Afer login, the host can be used by specifying the `--config <name>` on invocation

By default the alphabetically first host will be used if none are supplied. If a host is provided on the command line, and it is present in the config file, then the matching API key will be used.

## Connection Caching Daemon

By default, the tool will autospawn a temporary connection caching daemon to minimize the number of active connections required to a remote TrueNAS host

## Configuration

TrueNAS hosts and API keys can be stored in a JSON configuration file. The `config` commands can be used to modify this file.

```json
{
  "hosts":{
    "fangtooth":{
      "url":"wss://<servername>/api/current",
      "api_key":"api key goes here"
    },
    "other":{
      "url":"wss://<servername>/api/current",
      "api_key":"other api key"
    }
  }
}
```

The default path is `~/.truenas_incus_ctl/config.json`. It can be overridden with `--config-file`.

After a host has been added to the config-file, it can be specified with `--config <config name>`

## Run

`truenas_incus_ctl <command>`

`truenas_incus_ctl --host <truenas host> --api-key <truenas api key> <command>`

### Commands

- list
	- Print various datasets, snapshots and network shares
- dataset
	- Administer datasets/zvols and their associated shares
- replication
  - Perform replication tasks
- snapshot
	- Administer snapshots
- share
	- Administer network shares

## IPv6

When using IPv6 you must specify the IP address wrapped in `[]`, eg:

`truenas_incus_ctl --host '[aaaa:bbbb:cccc:dddd::1]' --allow-insecure --api-key $TN_APIKEY dataset ls`

If you need to specify a port:

`truenas_incus_ctl --host '[aaaa:bbbb:cccc:dddd::1]:8443' --allow-insecure --api-key $TN_APIKEY dataset ls`

## Testing

`go test -v ./cmd`

## Daemon Mode

During normal use `truenas_incus_ctl` will be launched in daemon mode with a 3 minute timeout. When updating the tool, be aware that the daemon will not refresh until the timeout expires, unless manually killed.

The deamon can normally be found in the process list with:

`ps -ax | grep "[t]ncdaemon"`

When debugging or testing, it can be useful to prelaunch the daemon to view its console. The follow command will launch the daemon as a foreground process without a timeout.

`sudo /home/<user>/go/bin/truenas_incus_ctl daemon /home/<user>/tncdaemon.sock`

## Middleware Patches

The following patches may be useful to support the Incus TrueNAS driver. 

- [back-port zfs.snapshot.rename functionality](#zfssnapshotrename-backport)
- [increase max_calls](#increase-max-api-calls)
- [iSCSI defer functionality](#iscsi-defer-functionality)

The middlewared source on a TrueNAS installation is located at `/usr/lib/python3/dist-packages/middlewared`

In order to apply patches to the middleware you need to [disable the read-only protection on the /usr dataset](#removing-usr-read-only-protection), and after applying the patch, [restart the middleware](#restarting-middlewared)

### Removing /usr read-only protection

In order to patch the middleware the current /usr dataset needs its readonly proection disabled.

The rootfs read-only protection can be removed by the following command: `zfs set readonly=off <active /usr dataset>`. You can use `zfs list /usr` to see the name of the active /usr dataset.

eg:

```sh
# zfs list /usr               
NAME                                                USED  AVAIL  REFER  MOUNTPOINT
boot-pool/ROOT/25.10.0-MASTER-20250525-015443/usr  3.06G  17.9G  3.04G  /usr
# zfs set readonly=off boot-pool/ROOT/25.10.0-MASTER-20250525-015443/usr
```

### zfs.snapshot.rename backport

The ability to rename a ZFS snapshot via the TrueNAS api is needed for full functionality of the Incus driver, and is present in 25.10 Goldeye nightlies, but can be added to previous versions relatively easily.

Firstly, [remove the /usr readonly protection](#removing-usr-read-only-protection), then edit the snapshot_actions.py file

`nano /usr/lib/python3/dist-packages/middlewared/plugins/zfs_/snapshot_actions.py`

to add the following function at the end of the file, if not already present

```python
    
    def rename(self, id_, new_name):
         try:
             with libzfs.ZFS() as zfs:
                 snapshot = zfs.get_snapshot(id_)
                 snapshot.rename(new_name)
         except libzfs.ZFSException as err:
             raise CallError(f'Failed to rename snapshot: {err}')
```

then [restart the middleware](#restarting-middlewared) 

### increase max-api calls

Although a connection-cache is used to limit the number of connections generated per IP, in some scenarios, eg automated containerized testing, it is possible to exceed the number of API calls allowed per IP in a finite period. This will result in persistant timeouts when it occurs. 

The maximum nummber of API calls allowed can be increased by modifiying the middleware's `max_calls` value, eg from 20 to 100.

disable the [/usr readonly protection](#removing-usr-read-only-protection), then apply the following patch, then [restart the middleware](#restarting-middlewared)
```diff
diff --git a/src/middlewared/middlewared/utils/rate_limit/cache.py b/src/middlewared/middlewared/utils/rate_limit/cache.py
index b04ecc1ec1..40797b71a7 100644
--- a/src/middlewared/middlewared/utils/rate_limit/cache.py
+++ b/src/middlewared/middlewared/utils/rate_limit/cache.py
@@ -12,7 +12,7 @@ __all__ = ['RateLimitCache']
 @dataclass(frozen=True)
 class RateLimitConfig:
     """The maximum number of calls per unique consumer of the endpoint."""
-    max_calls: int = 20
+    max_calls: int = 100
     """The maximum time in seconds that a unique consumer may request an
     endpoint that is being rate limited."""
     max_period: int = 60

```

### iSCSI Defer functionality ###

A pull-request for the TrueNAS Goldeye middleware has been submitted

https://github.com/truenas/middleware/pull/16614

The PR implements deferred iscsi reloading for iscsi target deletion, as well as defer capability on other iscsi commands

The iscsi target deletion improvement improves iscsi deletion operations by about 3.5x which makes a measurably improved performance
improvement.

When performance/integration testing the Incus driver, you may want to apply this patch if it is not already applied, as it can significantly reduce the runtime of integration tests.

### Restarting Middlewared

After making any patches to the middleware you will need to restart the middlewared service with `service middlewared restart`. This can take a few minutes.

