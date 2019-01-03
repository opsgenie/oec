package conf

func readConfigurationFromLocal(confPath string) (*Configuration, error) {
	return parseConfigurationFromFile(confPath)
}
