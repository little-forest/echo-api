# echo-api
Echo api server application for debugging and investigation.

## How to use

```
$ docker run --rm -p 8081:8081 littlef/echo-api:0.1.0 -p 8081
```

## How to build locally

```
$ goreleaser release --clean --snapshot
```
