package gomodcache

// StorageType as first variable.
// Cache dir as second variable.
// Format: https://raw.githubusercontent.com/gomods/athens/main/config.dev.toml.
const athensConfigFmt = `
GoBinary = "go"
GoEnv = "development"
GoBinaryEnvVars = ["GOPROXY=direct"]
GoGetWorkers = 30
ProtocolWorkers = 30
LogLevel = "error"
Timeout = 300
StorageType = "%s"
Port = ":3000"
CloudRuntime = "none"
RobotsFile = "robots.txt"
SumDBs = ["https://sum.golang.org"]
DownloadMode = "sync"
SingleFlightType = "memory"
IndexType = "none"

[Storage]
    [Storage.Disk]
        RootPath = "%s"
`
