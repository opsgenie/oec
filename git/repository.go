package git

import (
	"github.com/opsgenie/oec/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	"os"
	"sync"
)

type GitOptions struct {
	Url                string `json:"url" yaml:"url"`
	PrivateKeyFilepath string `json:"privateKeyFilepath" yaml:"privateKeyFilepath"`
	Passphrase         string `json:"passphrase" yaml:"passphrase"`
}

type GitUrl string

type Repositories map[GitUrl]*Repository

func NewRepositories() Repositories {
	return Repositories(make(map[GitUrl]*Repository))
}

func (r Repositories) NotEmpty() bool {
	return len(r) != 0
}

func (r Repositories) Get(gitUrl string) (*Repository, error) {
	if repository, contains := r[GitUrl(gitUrl)]; contains {
		return repository, nil
	}
	return nil, errors.Errorf("Git repository[%s] could not be found.", gitUrl)
}

func (r Repositories) DownloadAll(gitOptionsList []GitOptions) (err error) {

	for _, options := range gitOptionsList {
		err = r.Download(&options)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r Repositories) Download(options *GitOptions) (err error) {

	if _, contains := r[GitUrl(options.Url)]; !contains {
		repositoryPath, err := CloneMaster(options.Url, options.PrivateKeyFilepath, options.Passphrase)
		if err != nil {
			return errors.Errorf("Git repository[%s] could not be downloaded: %s", options.Url, err.Error())
		}

		logrus.Debugf("Git repository[%s] is downloaded.", options.Url)

		repository := NewRepository(repositoryPath, *options)
		err = repository.Chmod(0700)
		if err != nil {
			logrus.Warnf("Git repository[%s] chmod failed: %s", options.Url, err)
		}

		r[GitUrl(options.Url)] = repository
		return nil
	}

	logrus.Tracef("Git repository[%s] is already existed.", options.Url)
	return nil
}

func (r Repositories) PullAll() {
	for _, repository := range r {
		err := repository.Pull()
		if err == git.NoErrAlreadyUpToDate {
			logrus.Tracef("Git repository[%s] is already up-to-date.", repository.Options.Url)
			continue
		}
		if err != nil {
			logrus.Warnf("Git repository[%s] could not be pulled: %s", repository.Options.Url, err.Error())
			continue
		}
		logrus.Debugf("Git repository[%s] is pulled.", repository.Options.Url)
	}
}

func (r Repositories) RemoveAll() {
	for _, repository := range r {
		err := repository.Remove()
		if err != nil {
			logrus.Warnf("Git repository[%s] in directory[%s] could not be removed: %s", repository.Options.Url, repository.Path, err.Error())
		}
	}
}

/******************************************************************************************/

type Repository struct {
	Path    string
	Options GitOptions
	rw      *sync.RWMutex
}

func NewRepository(path string, options GitOptions) *Repository {
	return &Repository{
		rw:      &sync.RWMutex{},
		Path:    path,
		Options: options,
	}
}

func (r *Repository) Pull() error {
	r.rw.Lock()
	defer r.rw.Unlock()
	defer func() {
		err := util.ChmodRecursively(r.Path, 0700)
		if err != nil {
			logrus.Debugf("Git repository[%s] chmod failed: %s", r.Options.Url, err)
		}
	}()
	return PullMaster(r.Path, r.Options.PrivateKeyFilepath, r.Options.Passphrase)
}

func (r *Repository) Remove() error {
	r.rw.Lock()
	defer r.rw.Unlock()
	return os.RemoveAll(r.Path)
}

func (r *Repository) Chmod(mode os.FileMode) error {
	return util.ChmodRecursively(r.Path, mode)
}

func (r *Repository) RLock() {
	r.rw.RLock()
}

func (r *Repository) RUnlock() {
	r.rw.RUnlock()
}
