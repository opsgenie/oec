package conf

import (
	"os"
	"github.com/pkg/errors"
)

var Configuration map[string]string
var readConfigurationFromGitFunction = readConfigurationFromGit
var readConfigurationFromLocalFunction = readConfigurationFromLocal

func ReadConfFile() error {
	confSource := os.Getenv("MARIDCONFSOURCE")

	if confSource == "git" {
		username := os.Getenv("MARIDCONFREPOUSERNAME")
		password := os.Getenv("MARIDCONFREPOPASSWORD")
		gitUrl := os.Getenv("MARIDCONFREPOGITURL")

		gitConf, err := readConfigurationFromGitFunction(gitUrl, username, password)

		if err == nil {
			copied, err := cloneStringMap(gitConf)

			if err != nil {
				return err
			}

			Configuration = copied

			return nil
		} else {
			return err
		}
	} else if confSource == "local" {
		localConf, err := readConfigurationFromLocalFunction()

		if err == nil {
			copied, err := cloneStringMap(localConf)

			if err != nil {
				return err
			}

			Configuration = copied

			return nil
		} else {
			return err
		}
	} else {
		return errors.New("Unknown configuration source [" + confSource + "].")
	}
}

