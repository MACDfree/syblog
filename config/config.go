package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syblog/logger"

	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
)

type Config struct {
	SY   SYConfig   `toml:"siyuan"`
	Hugo HugoConfig `toml:"hugo"`
	SSH  SSHConfig  `toml:"ssh"`
}

type SYConfig struct {
	APIURL        string `toml:"apiURL"`
	APIToken      string `toml:"apiToken"`
	WorkspacePath string `toml:"workspacePath"`
	AssetsPath    string `toml:"-"`
}

type HugoConfig struct {
	ExcutePath  string `toml:"excutePath"`
	BlogPath    string `toml:"blogPath"`
	SectionName string `toml:"sectionName"`
}

type SSHConfig struct {
	Addr     string `toml:"addr"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	KeyPath  string `toml:"keyPath"`
	SitePath string `toml:"sitePath"`
}

var cfg Config

func GetConfig() Config {
	return cfg
}

func init() {
	file, err := os.Open("config.toml")
	if err != nil {
		logger.Fatalf("%+v", errors.Wrap(err, "配置文件config.toml打开失败"))
	}
	bs, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Fatalf("%+v", errors.Wrap(err, "配置文件config.toml读取失败"))
	}
	err = toml.Unmarshal(bs, &cfg)
	if err != nil {
		logger.Fatalf("%+v", errors.Wrap(err, "配置文件config.toml解析失败"))
	}

	if cfg.SY.APIURL == "" {
		cfg.SY.APIURL = "127.0.0.1:6806"
	}

	if cfg.SY.WorkspacePath == "" {
		logger.Fatalf("%+v", errors.New("workspacePath配置不能为空"))
	}

	if cfg.Hugo.BlogPath == "" {
		logger.Fatalf("%+v", errors.New("blogPath配置不能为空"))
	}

	if cfg.Hugo.SectionName == "" {
		cfg.Hugo.SectionName = "notes"
	}

	cfg.SY.AssetsPath = filepath.Join(cfg.SY.WorkspacePath, "data", "assets")
}
