package system

import (
	"io/ioutil"
	"log"
	"path/filepath"

	y "gopkg.in/yaml.v2"
)

// Yamlparser reads the snmp.yml file and unmarshalls it to a SNMPyaml struct
func Yamlparser() SNMPyaml {

	// Read the snmp.yml file
	absPath, _ := filepath.Abs("./app/snmp.yml")
	yamlFile, yamlerror := ioutil.ReadFile(absPath)
	if yamlerror != nil {
		log.Fatalf("ioutil err: %v", yamlerror)
	}

	/*
		c is the SNMPyaml struct
		yamlFile is unmarshalled to c
	*/

	var c SNMPyaml
	err := y.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Fatal(err)
	}

	return c
}

// ConfigParser reads the config.yml file and unmarshalls it to a Config struct
func ConfigParser() Config {

	// Read the snmp.yml file
	absPath, _ := filepath.Abs("./app/config.yml")
	yamlFile, yamlerror := ioutil.ReadFile(absPath)
	if yamlerror != nil {
		log.Fatalf("ioutil err: %v", yamlerror)
	}

	/*
		c is the Config struct
		c contains configuration for big-ip and datadog authentication
	*/

	var c Config
	err := y.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Fatal(err)
	}

	return c
}
