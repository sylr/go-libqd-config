package config

//go:generate deepcopy-gen --input-dirs . --output-package . --output-file-base config_deepcopy --go-header-file /dev/null
//+k8s:deepcopy-gen=true
//+k8s:deepcopy-gen:interfaces=sylr.dev/libqd/config.Config

// MyAppConfiguration implements sylr.dev/libqd/config.Config
type MyAppConfiguration struct {
	Reloads  int32  `yaml:"-"`
	Version  bool   `                                                       long:"version"`
	File     string `                                             short:"f" long:"config"`
	Verbose  []bool `yaml:"verbose" json:"verbose" toml:"verbose" short:"v" long:"verbose"`
	HTTPPort int    `yaml:"port"    json:"port"    toml:"port"    short:"p" long:"port"`
}

func (c *MyAppConfiguration) ConfigFile() string {
	return c.File
}
