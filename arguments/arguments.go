package arguments

import (
	"errors"

	"net/url"

	flag "github.com/spf13/pflag"
)

type Args struct {
	Cluster     string
	Endpoint    string
	Username    string
	Password    string
	Insecure    *bool
	Config      string
	CompactMode bool
}

func Parse() (*Args, error) {
	args := Args{}

	var insecure bool

	flag.StringVarP(&args.Cluster, "cluster", "c", "", "the cluster to select from the configuration")
	flag.StringVarP(&args.Endpoint, "endpoint", "e", "", "the cluster endpoint to query (takes precedence over cluster)")
	flag.StringVarP(&args.Username, "username", "u", "", "the username to use for endpoint authentication if provided as argument or none is specified in the configuration")
	flag.StringVarP(&args.Password, "password", "p", "", "the pssword to use for endpoint authentication if provided as argument or none is specified in the configuration")
	flag.BoolVarP(&insecure, "insecure", "k", false, "the pssword to use for endpoint authentication if provided as argument or none is specified in the configuration")
	flag.StringVarP(&args.Config, "config", "f", "", "the configuration file to use")
	flag.BoolVarP(&args.CompactMode, "compact", "m", false, "compact mode (shows only cluster overview):")

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

	insecureFlag := flag.Lookup("insecure")
	if !insecureFlag.Changed {
		args.Insecure = nil
	} else {
		args.Insecure = &insecure
	}

	return &args, nil
}
