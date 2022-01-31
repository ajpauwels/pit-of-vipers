# A pit of vipers

## Objective

Do you use golang? Do you use viper for configuration management
within golang? Are you sick and tired of:

- Only handling one config file at a time

- Being thread-unsafe

- Having to write dodgy synchronization code to merge multiple config
  files from multiple source deterministically

- Never going through the effort of live-reloading config on
  filesystem updates

Look no more! Pit of Vipers ingests as many viper instances as you
want and:

- Deterministically merges them in the order provided

- Updates the merged config every time a sub-instance receives an
  update from the filesystem
  
- Forces use of live-reloaded config

- Does all of this in a thread-safe manner

## How to

TL;DR:

```golang
package main

import (
	"fmt"
	viperpit "github.com/ajpauwels/pit-of-vipers"
)

type Config struct {
	Host      string `mapstructure:"host"`
	Port      uint16 `mapstructure:"port"`
	SecretKey string `mapstructure:"secretkey"`
	NewKey    string `mapstructure:"newkey"`
}

func main() {
	vpCh, errCh := viperpit.NewFromPaths([]string{"./config", "./config/shared", "./config/app", "./config/preview/shared", "./config/preview/app"}) // (1)
	for { // (2)
		select {
		case vp := <-vpCh: // (3)
			var config Config
			vp.Unmarshal(&config)
			fmt.Printf("%+v\n", config)
		case err := <-errCh: // (4)
			fmt.Errorf("%s", err)
		}
	}
}
```

Explanation of highlighted portions below:

1. Calling a `New*` function on `viperpit` returns two channels: the viper
   channel which receives a merged viper instance every time one of
   the sub-instances is updated, and an error channel which receives
   all errors which occurred in this process

2. Main thread just loops infinitely on a channel select statement,
   waiting for config updates or errors in the config update process
   
3. The `<-vpCh` case receives a fully merged viper instance every time
   one of the sub-instances is updated from the filesystem
   
4. The `<-errCh` case receives any errors that may have occurred during
   the merging process
