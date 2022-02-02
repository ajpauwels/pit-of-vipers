package config

import (
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

/// ViperPit stores many viper instances and merges them into one
type ViperPit struct {
	rwConfigs sync.Mutex
	vipers    []*viper.Viper
	configs   []map[string]interface{}
}

/// NewFromPathsAndName takes as input an array of paths and the name
/// of the config file (without extension) potentially stored at those
/// paths and creates a ViperPit that monitors each available config.
func NewFromPathsAndName(paths []string, name string) (viperChannel chan *viper.Viper, errChannel chan error) {
	// Make the array that'll store all our viper instances
	vipers := make([]*viper.Viper, len(paths))

	// Each viper instance looks at the given path for a file named
	// with the given name with any supported extension
	for i, path := range paths {
		v := viper.New()
		v.AddConfigPath(path)
		v.SetConfigName(name)
		vipers[i] = v
	}

	return New(vipers)
}

/// NewFromPaths takes as input an array of paths and creates a
/// ViperPit that monitors each available file named config.* at those
/// paths.
func NewFromPaths(paths []string) (viperChannel chan *viper.Viper, errChannel chan error) {
	// Make the array that'll store all our viper instances
	vipers := make([]*viper.Viper, len(paths))

	// Each viper instance looks at the given path for a file named
	// config with any supported extension
	for i, path := range paths {
		v := viper.New()
		v.AddConfigPath(path)
		v.SetConfigName("config")
		vipers[i] = v
	}

	return New(vipers)
}

/// NewFromPaths takes as input an array of vipers and creates a
/// ViperPit that monitors and merges and each one
func New(vipers []*viper.Viper) (viperChannel chan *viper.Viper, errChannel chan error) {
	// Initialize our viper pit
	base := viper.New()
	pit := &ViperPit{
		vipers:  vipers,
		configs: make([]map[string]interface{}, len(vipers)),
	}

	// Initialize our channels
	viperChannel = make(chan *viper.Viper)
	errChannel = make(chan error)

	// Read and setup each config
	for i, v := range vipers {
		// Ingest config
		err := v.ReadInConfig()
		if err != nil {
			go func() { errChannel <- err }()
		} else {
			base.MergeConfigMap(v.AllSettings())

			// If the config file changes, atomically update the shared
			// config state for that config instance and notify the
			// channel
			viperIndex := i
			v.OnConfigChange(func(in fsnotify.Event) {
				// Lock access the configs slice
				pit.rwConfigs.Lock()
				defer pit.rwConfigs.Unlock()

				// Fetch the viper that was updated
				v := pit.vipers[viperIndex]

				// If the viper is non-nil, re-compute config
				if v != nil {
					// Fetch the viper's config set
					pit.configs[viperIndex] = v.AllSettings()

					// Create a temporary viper instance that will
					// store the updated config computation
					sumViper := viper.New()

					// Compute config into sumViper
					for i := 0; i < len(pit.configs); i++ {
						if pit.configs[i] != nil {
							err := sumViper.MergeConfigMap(pit.configs[i])
							if err != nil {
								go func() { errChannel <- err }()
							}
						}
					}

					// Merge the newly computed config with the
					// existing config
					err := base.MergeConfigMap(sumViper.AllSettings())
					if err != nil {
						go func() { errChannel <- err }()
					} else {
						// Copy the newly computed config and send it
						// over the channel
						returnedViper := viper.New()
						err := returnedViper.MergeConfigMap(base.AllSettings())
						if err != nil {
							go func() { errChannel <- err }()
						} else {
							returnedViper.AutomaticEnv()
							viperChannel <- returnedViper
						}
					}
				}
			})

			// Activate configuration file watching
			defer v.WatchConfig()
		}
	}

	// Pass first completed set of configuration to consuming app
	go func() {
		returnedViper := viper.New()
		err := returnedViper.MergeConfigMap(base.AllSettings())
		if err != nil {
			go func() { errChannel <- err }()
		} else {
			returnedViper.AutomaticEnv()
			viperChannel <- returnedViper
		}
	}()

	return
}
