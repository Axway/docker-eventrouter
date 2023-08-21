

# 0.2.0 (dev)
- BREAKING-CHANGE : refactor streams to add `reader`, `transforms`, `writer` section
- BREAKING-CHANGE : connector now are identified through `type` key instead of `name`
- use extension `.ser.yml` to enable schema validation from `qlt-router-schema.yml` (under `vscode`)

# 0.0.1sink
- connector: `file`, add `MaxFile`, `MaxSize` for automatic file rotation and jsonfile sample
- add `list-connectors` `list-config` commands (in addition to `help`, `version`)
- add preliminary support for AWS SQS
- add `--port` for basic ui/prometheus/health 
- add `--config` option
- add `--memprof` `--cpuprof` options
- verify configuration file connectors attribute, fail to start if not found
- remove nasral package (need to find an alternative for env variable)
- add a new log module to support traceability (incomplete)
