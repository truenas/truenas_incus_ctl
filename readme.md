# truenas-admin

`truenas-admin` is a tool for administering datasets, snapshots and network shares that are hosted on a TrueNAS server.

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
      "api_key":"other api key",
    }
  }
}
```

The default path is `~/.truenas-admin/config.json`. It can be overridden with --config.

## Run

`truenas-admin --url <websocket server> --api-key <api key> <command>`

### Commands

- list
	- Print various datasets, snapshots and network shares
- dataset
	- Administer datasets/zvols and their associated shares
- snapshot
	- Administer snapshots
- share
	- Administer network shares

## Testing

`go test -v ./cmd`

