module github.com/daison12006013/turboscript

go 1.23.10

require (
	github.com/bradfitz/gomemcache v0.0.0-20250403215159-8d39553ac7cf
	github.com/daison12006013/turboscript/turbo_modules/argon2 v0.0.0-00010101000000-000000000000
	github.com/dop251/goja v0.0.0-20250630131328-58d95d85e994
	github.com/dop251/goja_nodejs v0.0.0-20250409162600-f7acab6894b0
	github.com/evanw/esbuild v0.25.5
	github.com/fasthttp/router v1.5.4
	github.com/fasthttp/websocket v1.5.12
	github.com/lib/pq v1.10.9
	github.com/minio/minio-go/v7 v7.0.59
	github.com/redis/go-redis/v9 v9.11.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/russross/blackfriday/v2 v2.1.0
	github.com/segmentio/kafka-go v0.4.48
	github.com/stretchr/testify v1.8.0
	github.com/valyala/fasthttp v1.62.0
	golang.org/x/crypto v0.40.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/dop251/base64dec v0.0.0-20231022112746-c6c9f9a96217 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/google/pprof v0.0.0-20241029153458-d1b30febd7db // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/savsgio/gotils v0.0.0-20240704082632-aef3928b8a38 // indirect
	github.com/sirupsen/logrus v1.9.2 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)

replace github.com/daison12006013/turboscript/turbo_modules/argon2 => ./turbo_modules/argon2

replace github.com/daison12006013/turboscript/turbo_modules/mysql2 => ./turbo_modules/mysql2

replace github.com/daison12006013/turboscript/turbo_modules/pg => ./turbo_modules/pg
