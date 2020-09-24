package main

import (
	"os"

	"github.com/gobwas/glob"
	"github.com/imdario/mergo"
	"github.com/kovetskiy/ko"
	"github.com/reconquest/hierr-go"
	"gopkg.in/yaml.v2"
)

type FileConfig struct {
	Pull struct {
		Format string `yaml:"format,omitempty"`
	} `yaml:"pull,omitempty"`

	Push struct {
		Type       string            `yaml:"type,omitempty"`
		Directives map[string]string `yaml:"directives,omitempty,flow"`
	} `yaml:"push,omitempty"`
}

type LocaleConfig struct {
	Application string `yaml:"application",required:"true"`
	Smartling   string `yaml:"smartling",required:"true"`
}

type Config struct {
	UserID    string `yaml:"user_id",required:"true"`
	Secret    string `yaml:"secret",required:"true"`
	AccountID string `yaml:"account_id"`
	ProjectID string `yaml:"project_id,omitempty"`
	Threads   int    `yaml:"threads"`

	Locales   []LocaleConfig `yaml:"locales"`
	LocaleToAppLocaleMap   map[string]string
	AppLocaleToLocaleMap   map[string]string

	Files map[string]FileConfig `yaml:"files"`

	Proxy string `yaml:"proxy,omitempty"`

	path string
}

func NewConfig(path string) (Config, error) {
	config := Config{
		path: path,
	}

	err := ko.Load(path, &config, yaml.Unmarshal)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}

		return config, err
	}

	if (config.Locales != nil && len(config.Locales) > 0) {
		config.AppLocaleToLocaleMap = make(map[string]string)
		config.LocaleToAppLocaleMap = make(map[string]string)
		for _, locale := range config.Locales {
			config.AppLocaleToLocaleMap[locale.Application] = locale.Smartling
			config.LocaleToAppLocaleMap[locale.Smartling] = locale.Application
		}
	}

	return config, nil
}

func (config *Config) GetFileConfig(path string) (FileConfig, error) {
	var (
		match FileConfig
		found bool
	)

	for key, candidate := range config.Files {
		pattern, err := glob.Compile(key, '/')
		if err != nil {
			return FileConfig{}, NewError(
				hierr.Errorf(
					err,
					`unable to compile pattern from config file (key "%s")`,
					key,
				),

				`File match pattern is malformed. Check out help for more `+
					`information on globbing patterns.`,
			)
		}

		if pattern.Match(path) {
			match = candidate
			found = true
		}
	}

	defaults := config.Files["default"]

	if !found {
		return defaults, nil
	}

	err := mergo.Merge(&match, defaults)
	if err != nil {
		return FileConfig{}, NewError(
			hierr.Errorf(err, "unable to merge file config options"),
			`It's internal error. Consider reporting bug.`,
		)
	}

	return match, nil
}
