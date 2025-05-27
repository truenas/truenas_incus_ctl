# truenas_incus_ctl

`truenas_incus_ctl` is a tool for administering datasets, snapshots and network shares that are hosted on a TrueNAS server.

## Install

`go build`

`go install`

## Configuration

TrueNAS hosts can be specified with a JSON configuration file.

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

The default path is `~/.truenas_incus_ctl/config.json`. It can be overridden with --config.

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

The following patches to middlewared are currently needed to support the Incus TrueNAS driver. After making the patches you will need to restart the middlewared service
with `service middlewared restart`

The middlewared source on a TrueNAS installation is located at `/usr/lib/python3/dist-packages/middlewared`

Modify `zfs.dataset.rename` to support snapshots. The snapshot rename capability is required by Incus.

```diff
diff --git a/src/middlewared/middlewared/plugins/zfs_/dataset_actions.py b/src/middlewared/middlewared/plugins/zfs_/dataset_actions.py
index 5dc1780fd8..76c57b554c 100644
--- a/src/middlewared/middlewared/plugins/zfs_/dataset_actions.py
+++ b/src/middlewared/middlewared/plugins/zfs_/dataset_actions.py
@@ -83,10 +83,11 @@ class ZFSDatasetService(Service):
             Bool('recursive', default=False)
         )
     )
+
     def rename(self, name, options):
         try:
             with libzfs.ZFS() as zfs:
-                dataset = zfs.get_dataset(name)
+                dataset = zfs.get_object(name)
                 dataset.rename(options['new_name'], recursive=options['recursive'])
         except libzfs.ZFSException as e:
             self.logger.error('Failed to rename dataset', exc_info=True)
```

Although a connection-cache is used to limit the number of connections generated per IP, in some scenarios, eg automated containerized testing, it is possible to exceed the number of API calls allowed per IP in a finite period. This will result in persistant timeouts when it occurs. 

The maximum nummber of API calls allowed can be increased by modifiying the middleware's `max_calls` value, eg from 20 to 100.

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

The rootfs read-only protection can be removed by the following command: `zfs set readonly=off <BOOT-POOL>/ROOT/<TRUENAS-BOOT-ENV>/usr`, where `<TRUENAS-BOOT-ENV>` is the name of the active TrueNAS Boot Environment, eg: "25.10.0", or "25.10.0-MASTER-20250519-015438, and `<BOOT-POOL>` is the name of the boot pool, eg: "boot-pool". You can use `cat /proc/mounts | grep /usr` to see the name of the active /usr dataset