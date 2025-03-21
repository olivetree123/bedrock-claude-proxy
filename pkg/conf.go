package pkg

import (
	log "bedrock-claude-proxy/log"
	"bytes"
	"encoding/json"
	"os"
)

type Config struct {
	HttpConfig
	BedrockConfig *BedrockConfig `json:"bedrock_config,omitempty"`
}

func NewConfigFromLocal(filename string) (*Config, error) {
	conf := &Config{}
	err := conf.load(filename)
	return conf, err
}

func (this *Config) MarginWithENV() {
	if len(this.WebRoot) <= 0 {
		this.WebRoot = os.Getenv("WEB_ROOT")
	}
	if len(this.Listen) <= 0 {
		this.Listen = os.Getenv("HTTP_LISTEN")
	}
	if len(this.APIKey) <= 0 {
		this.APIKey = os.Getenv("API_KEY")
	}
	if this.BedrockConfig == nil {
		this.BedrockConfig = LoadBedrockConfigWithEnv()
	}
}

func (c *Config) load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Logger.Error(err)
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(c)
	if err != nil {
		log.Logger.Error(err)
	}
	return err
}

func (c *Config) ToJSON() (string, error) {
	jsonBin, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	var str bytes.Buffer
	_ = json.Indent(&str, jsonBin, "", "  ")
	return str.String(), nil
}

func (c *Config) Save(saveAs string) error {
	file, err := os.Create(saveAs)
	if err != nil {
		log.Logger.Error(err)
		return err
	}
	defer file.Close()
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		log.Logger.Error(err)
		return err
	}
	_, err = file.Write(data)
	if err != nil {
		log.Logger.Error(err)
	}
	return err
}
