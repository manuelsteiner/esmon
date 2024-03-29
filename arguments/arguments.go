package arguments

import (
	"errors"

	flag "github.com/spf13/pflag"
	"net/url"
)

type Args struct {
	Cluster  string
	Endpoint string
	Username string
	Password string
	Config   string
}

func Parse() (*Args, error) {
	args := Args{}

	flag.StringVarP(&args.Cluster, "cluster", "c", "", "the cluster to select from the configuration")
	flag.StringVarP(&args.Endpoint, "endpoint", "e", "", "the cluste endpoint to query (takes precedence over cluster)")
	flag.StringVarP(&args.Username, "username", "u", "", "the username to use for endpoint authentication if provided as argument or none is specified in the configuration")
	flag.StringVarP(&args.Password, "password", "p", "", "the pssword to use for endpoint authentication if provided as argument or none is specified in the configuration")
	flag.StringVarP(&args.Config, "config", "f", "", "the configuration file to use")

	flag.Parse()

	if args.Endpoint != "" {
		url, err := url.Parse(args.Endpoint)
		if err != nil || url.Scheme == "" || url.Host == "" {
			return nil, errors.New("Endpoint must be an URL.")
		}

		if args.Username == "" || args.Password == "" {
			return nil, errors.New("Credentials (Username, Password) must be used when specifying Endpoint.")
		}
	}

	if args.Username != "" && args.Password == "" {
		return nil, errors.New("Password must be used when specifying Username.")
	}

	return &args, nil
}
