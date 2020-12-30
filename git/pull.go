package git

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
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

	options := &git.PullOptions{
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		ReferenceName:     plumbing.Master,
		SingleBranch:      true,
		Force:             true,
	}

	if privateKeyFilePath != "" {

		auth, err := ssh.NewPublicKeysFromFile(ssh.DefaultUsername, privateKeyFilePath, passPhrase)
		if err != nil {
			return err
		}

		options.Auth = auth
	}

	return w.Pull(options)
}

func FetchAndReset(repositoryPath, privateKeyFilePath, passPhrase string) error {

	r, err := git.PlainOpen(repositoryPath)
	if err != nil {
		return err
	}

	options := &git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/heads/master:refs/heads/master"},
	}

	if privateKeyFilePath != "" {

		auth, err := ssh.NewPublicKeysFromFile(ssh.DefaultUsername, privateKeyFilePath, passPhrase)
		if err != nil {
			return err
		}

		options.Auth = auth
	}

	err = r.Fetch(options)
	if err != nil {
		return err
	}

	ref, err := r.Reference(plumbing.Master, true)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	return w.Reset(&git.ResetOptions{
		Commit: ref.Hash(),
		Mode:   git.HardReset,
	})
}
