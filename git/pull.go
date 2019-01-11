package git

import (
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

func Pull(repositoryPath, privateKeyFilePath, passPhrase string) error {
	r, err := git.PlainOpen(repositoryPath)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	opts, err := getPullOptions(privateKeyFilePath, passPhrase)
	if err != nil {
		return err
	}

	return w.Pull(opts)
}

func getPullOptions(privateKeyFilePath, passPhrase string) (*git.PullOptions, error) {

	opts := &git.PullOptions {
		RemoteName: "origin",
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}

	if privateKeyFilePath != "" {

		auth, err := ssh.NewPublicKeysFromFile(ssh.DefaultUsername, privateKeyFilePath, passPhrase)
		if err != nil {
			return nil, err
		}

		opts.Auth = auth
	}

	err := opts.Validate()
	if err != nil {
		return nil, err
	}

	return opts, nil
}
