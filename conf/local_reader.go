package conf

func readFileFromLocal(filepath string) (*Configuration, error) {

	err := checkFileExtension(filepath)
	if err != nil {
		return nil, err
	}

	return readFile(filepath)
}
