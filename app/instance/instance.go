package instance

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// VERSION версия программы
const VERSION = "0.01"

// Config парамтеры из конфигурационного файла программы.
type Config struct {
	Main struct {
		Confd    string `yaml:"confd"`
		Loglevel string `yaml:"loglevel"`
	} `yaml:"main"`
}

// Instance ...
type Instance struct {
	logs   *logrus.Logger
	config *Config
}

// NewInstance ...
func NewInstance() *Instance {
	var configFileName string
	showVersion := false

	flag.StringVar(&configFileName, "c", "conf.yaml", "Config file")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.Parse()

	if showVersion {
		fmt.Print("Version: ", VERSION, "\n")
		return nil
	}

	config, err := ReadConfigFile(configFileName)
	if err != nil {
		fmt.Print("Не могу загрузить информацию из конфигурационного файла. ", err)
	}

	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})

	// Устанавливаем уровень важности сообщений, выводимых logrus
	level, err := logrus.ParseLevel(config.Main.Loglevel)
	if err != nil {
		logs.Warn("Не правильно определён loglevel в конфигурационном файле. Установлен уровень по умолчанию debug.")
		level, _ = logrus.ParseLevel("debug")
	}
	logs.SetLevel(level)

	return &Instance{
		logs:   logs,
		config: config,
	}
}

// ReadConfigFile Читаем конфигурацию из конфигурационного файла и формируем
// структуру Conf
func ReadConfigFile(fileName string) (*Config, error) {
	// Чтение конфигурационного файла
	confFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var conf Config
	err = yaml.Unmarshal(confFile, &conf)
	if err != nil {
		return nil, err
	}

	return &conf, nil
}
