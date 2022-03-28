package main

import (
	"io/ioutil"
	"log"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

var conf *Configure
var WorkDirInRootfs, WorkDirGlobal string
var DataFilesPath, CacheFilesPath string

type Configure struct {
	Rootfs         RootfsConfig `yaml:"rootfs"`
	MQ             MQConfig     `yaml:"mq"`
	DataFilesPath  string       `yaml:"dataFilesPath"`
	CacheFilesPath string       `yaml:"cacheFilesPath"`
}

type RootfsConfig struct {
	RootfsPath         string `yaml:"rootfsPath"`
	ContainerFilesPath string `yaml:"containerFilesPath"`
	WorkDir            string `yaml:"workDir"`
	WorkUser           string `yaml:"workUser"`
}

type MQConfig struct {
	IP       string `yaml:"ip"`
	Port     int    `yaml:"port"`
	UserName string `yaml:"userName"`
	Password string `yaml:"password"`
}

func InitConfig(filePath *string) {
	fileBytes, err := ioutil.ReadFile(*filePath)
	if err != nil {
		log.Println("[FAILED] read config file failed")
		panic(err)
	}
	if err = yaml.Unmarshal(fileBytes, &conf); err != nil {
		log.Println("[FAILED] unmarshal yaml file failed")
		panic(err)
	}
	WorkDirInRootfs = conf.Rootfs.WorkDir
	WorkDirGlobal = filepath.Join(conf.Rootfs.RootfsPath, WorkDirInRootfs)
	DataFilesPath = conf.DataFilesPath
	CacheFilesPath = conf.CacheFilesPath
	log.Println("[INFO] Init config successfully")
}
