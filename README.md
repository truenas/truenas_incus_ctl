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

`truenas_incus_ctl --url <websocket server> --api-key <api key> <command>`

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

## Testing

`go test -v ./cmd`

## Middleware Patches

The following patches may be useful to support the Incus TrueNAS driver. 

- [back-port zfs.snapshot.rename functionality](#zfssnapshotrename-backport)
- [increase max_calls](#increase-max-api-calls)

The middlewared source on a TrueNAS installation is located at `/usr/lib/python3/dist-packages/middlewared`

In order to apply patches to the middleware you need to [disable the read-only protection on the /usr dataset](#removing-usr-read-only-protection), and after applying the patch, [restart the middleware](#restarting-middlewared)

### Removing /usr read-only protection

In order to patch the middleware the current /usr dataset needs its readonly proection disabled.

The rootfs read-only protection can be removed by the following command: `zfs set readonly=off <BOOT-POOL>/ROOT/<TRUENAS-BOOT-ENV>/usr`, where `<TRUENAS-BOOT-ENV>` is the name of the active TrueNAS Boot Environment, eg: "25.10.0", or "25.10.0-MASTER-20250519-015438, and `<BOOT-POOL>` is the name of the boot pool, eg: "boot-pool". You can use `cat /proc/mounts | grep /usr` to see the name of the active /usr dataset

eg:

```sh
# zfs list | grep /usr
boot-pool/ROOT/25.04.0/usr                                                                                                      2.51G  37.5G  2.51G  /usr
boot-pool/ROOT/25.10.0-MASTER-20250525-015443/usr                                                                               2.86G  37.5G  2.86G  /usr
# zfs set readonly=off boot-pool/ROOT/25.10.0-MASTER-20250525-015443/usr
```

### zfs.snapshot.rename backport

The ability to rename a ZFS snapshot via the TrueNAS api is needed for full functionality of the Incus driver. Currently this support is not included in the middleware, but can be added relatively easily.

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

### Restarting Middlewared

After making any patches to the middleware you will need to restart the middlewared service with `service middlewared restart`. This can take a few minutes.

