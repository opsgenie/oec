package git

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
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

func FetchAndReset(repositoryPath, privateKeyFilePath, passPhrase, branch string) error {

	r, err := git.PlainOpen(repositoryPath)
	if err != nil {
		return err
	}

	if branch == "" {
		branch = "master"
	}

	refSpec := config.RefSpec("+refs/heads/" + branch + ":refs/remotes/origin/" + branch)
	options := &git.FetchOptions{
		RefSpecs: []config.RefSpec{refSpec},
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

	remoteRef := plumbing.NewRemoteReferenceName("origin", branch)
	ref, err := r.Reference(remoteRef, true)
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
