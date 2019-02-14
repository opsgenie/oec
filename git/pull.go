package git

import (
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

func PullMaster(repositoryPath, privateKeyFilePath, passPhrase string) error {
	r, err := git.PlainOpen(repositoryPath)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	return gitPullMaster(w, privateKeyFilePath, passPhrase)
}

func gitPullMaster(w *git.Worktree, privateKeyFilePath, passPhrase string) error {

	options := &git.PullOptions {
		RecurseSubmodules: 	git.DefaultSubmoduleRecursionDepth,
		ReferenceName: 		plumbing.Master,
		SingleBranch:  		true,
	}

	if privateKeyFilePath != "" {

		auth, err := ssh.NewPublicKeysFromFile(ssh.DefaultUsername, privateKeyFilePath, passPhrase)
		if err != nil {
			return err
		}

		options.Auth = auth
	}

	err := options.Validate()
	if err != nil {
		return err
	}

	return w.Pull(options)
}
