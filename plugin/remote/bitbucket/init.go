package bitbucket

import (
	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/shared/model"
)

func init() {
	remote.Register(model.RemoteBitbucket, plugin)
}

func plugin(remote *model.Remote) remote.Remote {
	return &Bitbucket{
		URL:     remote.URL,
		API:     remote.API,
		Client:  remote.Client,
		Secret:  remote.Secret,
		Enabled: remote.Open,
	}
}
