package main

import (
	"io/ioutil"
	"log"

	"github.com/spf13/viper"
)

// Create a default config file if not found config.toml
func loadConfigIfExists() {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("toml")   // REQUIRED if the config file does not have the extension in the name
	//viper.AddConfigPath("/etc/rsync-os/")   // path to look for the config file in
	//viper.AddConfigPath("$HOME/.rsync-os")  // call multiple times to add many search paths
	viper.AddConfigPath(".") // optionally look for config in the working directory

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config File not found

			createSampleConfig()
			log.Fatalln("Config does not exist, a sample of config was created")
		} else {
			// Found but got errors
			log.Fatalln(err)
		}
	}
}

func createSampleConfig() {
	confSample := []byte(
		`title = "configuration of mirrors-os"

[global]
  server = "127.0.0.1:52111"

[archlinux]
  name = "archlinux"
  src = "rsync://mirrors.tuna.tsinghua.edu.cn/archlinux/"
  srcroot = "rsync://mirrors.tuna.tsinghua.edu.cn/archlinux/"
  cookie = "TEAMBITION_SESSIONID=xxx; TEAMBITION_SESSIONID.sig=xxx"
  base = "mirrors/"
  dbpath = "archlinux.db"
  cron = "0 0 0,12 * * *"

[archlinuxcn]
  name = "archlinuxcn"
  src = "rsync://mirrors.tuna.tsinghua.edu.cn/archlinuxcn/"
  srcroot = "rsync://mirrors.tuna.tsinghua.edu.cn/archlinuxcn/"
  cookie = "TEAMBITION_SESSIONID=xxx; TEAMBITION_SESSIONID.sig=xxx"
  base = "mirrors/"
  dbpath = "archlinuxcn.db"
  cron = "0 0 6,18 * * *"
`)

	if ioutil.WriteFile("config.toml", confSample, 0666) != nil {
		log.Fatalln("Can't create a sample of config")
	}

}
