package conf

func readConfigurationFromLocal(confFilepath string) (*Configuration, error) {

	err := checkFileExtension(confFilepath)
	if err != nil {
		return nil, err
	}

	return readConfigurationFromFile(confFilepath)
}
