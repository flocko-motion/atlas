package cli

import "rankedb/apiclient"

func Client() (*apiclient.ClientWithResponses, error) {
	return apiclient.NewClientWithResponses(Cfg.Server)
}
