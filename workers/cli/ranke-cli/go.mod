module github.com/flocko-motion/ranke-cli

go 1.26.2

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/spf13/cobra v1.10.2
	rankedb v0.0.0
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/oapi-codegen/runtime v1.4.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
)

replace rankedb => ../../../rankedb/go
