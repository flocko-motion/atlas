package cli

import "github.com/flocko-motion/rankedb/go/apiclient"

func Client() (*apiclient.ClientWithResponses, error) {
	return apiclient.NewClientWithResponses(Cfg.Server)
}
