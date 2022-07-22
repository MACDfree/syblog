package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"syblog/logger"

	"github.com/pkg/errors"
)

type Config struct {
	SiYuanAPI     string `json:"siyuanAPI"`
	WorkspacePath string `json:"workspacePath"`
	HugoGenPath   string `json:"hugoGenPath"`
	AssetsPath    string `json:"-"`
}

var cfg Config

func GetConfig() Config {
	return cfg
}

func init() {
	file, err := os.Open("config.json")
	if err != nil {
		logger.Fatalf("%+v", errors.Wrap(err, "配置文件config.json打开失败"))
	}
	bs, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Fatalf("%+v", errors.Wrap(err, "配置文件config.json读取失败"))
	}
	err = json.Unmarshal(bs, &cfg)
	if err != nil {
		logger.Fatalf("%+v", errors.Wrap(err, "配置文件config.json解析失败"))
	}

	if cfg.SiYuanAPI == "" {
		cfg.SiYuanAPI = "127.0.0.1:6806"
	}

	if cfg.WorkspacePath == "" {
		logger.Fatalf("%+v", errors.New("workspacePath配置不能为空"))
	}

	if cfg.HugoGenPath == "" {
		logger.Fatalf("%+v", errors.New("hugoGenPath配置不能为空"))
	}

	cfg.AssetsPath = filepath.Join(cfg.WorkspacePath, "assets")
}
