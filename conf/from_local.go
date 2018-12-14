package conf

func readConfigurationFromLocal(confPath string) (*Configuration, error) {
	return parseConfiguration(confPath)
}
