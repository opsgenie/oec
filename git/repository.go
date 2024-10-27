package git

import (
	"os"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/opsgenie/oec/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Options struct {
	Url                string `json:"url" yaml:"url"`
	PrivateKeyFilepath string `json:"privateKeyFilepath" yaml:"privateKeyFilepath"`
	Passphrase         string `json:"passphrase" yaml:"passphrase"`
	Branch             string `json:"branch" yaml:"branch"`
}

type Url string

type Repositories map[Url]*Repository

func NewRepositories() Repositories {
	return make(map[Url]*Repository)
}

func (r Repositories) NotEmpty() bool {
	return len(r) != 0
}

func (r Repositories) Get(url string) (*Repository, error) {
	if repository, contains := r[Url(url)]; contains {
		return repository, nil
	}
	return nil, errors.Errorf("Git repository[%s] could not be found.", url)
}

func (r Repositories) DownloadAll(optionsList []Options) (err error) {

	for _, options := range optionsList {
		err = r.Download(&options)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r Repositories) Download(options *Options) (err error) {

	if _, contains := r[Url(options.Url)]; !contains {
		repositoryPath, err := CloneMaster(options.Url, options.PrivateKeyFilepath, options.Passphrase, options.Branch)
		if err != nil {
			return errors.Errorf("Git repository[%s] could not be downloaded: %s", options.Url, err.Error())
		}

		logrus.Debugf("Git repository[%s] is downloaded.", options.Url)

		r[Url(options.Url)] = NewRepository(repositoryPath, *options)
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
	Options Options
	rw      *sync.RWMutex
}

func NewRepository(path string, options Options) *Repository {
	repository := &Repository{
		rw:      &sync.RWMutex{},
		Path:    path,
		Options: options,
	}

	err := repository.Chmod(0700)
	if err != nil {
		logrus.Warnf("Git repository[%s] chmod failed: %s", options.Url, err)
	}

	return repository
}

func (r *Repository) Pull() error {
	r.rw.Lock()
	defer r.rw.Unlock()
	defer func() {
		err := util.ChmodRecursively(r.Path, 0700)
		if err != nil {
			logrus.Warnf("Git repository[%s] chmod failed: %s", r.Options.Url, err)
		}
	}()
	return FetchAndReset(r.Path, r.Options.PrivateKeyFilepath, r.Options.Passphrase, r.Options.Branch)
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
