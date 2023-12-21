package config

import (
	"os"

	"github.com/ubtr/ubt-go/commons"

	yaml "gopkg.in/yaml.v3"
)

type ChainTypeConfig struct {
	Networks       map[string]ChainConfig `yaml:"networks"`
	DefaultNetwork string                 `yaml:"defaultNetwork"`
}

type ChainConfig struct {
	Testnet      bool    `yaml:"testnet"`
	LimitRPS     float64 `yaml:"limitRPS"`
	ChainType    string  `yaml:"-"`
	ChainNetwork string  `yaml:"-"`
	RpcUrl       string  `yaml:"rpcUrl"`
}

type Config struct {
	LimitRPS float64                    `yaml:"limitRPS"`
	Chains   map[string]ChainTypeConfig `yaml:"chains"`
}

func (c *Config) GetChainConfig(strChainId string) *ChainConfig {
	chainId := commons.StringToChainId(strChainId)
	conf := c.Chains[chainId.Type].Networks[chainId.Network]
	conf.ChainType = chainId.Type
	conf.ChainNetwork = chainId.Network
	return &conf
}

func LoadConfig(path string) *Config {
	var conf Config
	f, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(f, &conf)
	if err != nil {
		panic(err)
	}
	return &conf
}
