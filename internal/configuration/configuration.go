package configuration

import (
	"os"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"
)

type Configuration struct {
	URL      string
	Token    string
	Insecure bool
}

func New() (Configuration, error) {
	return NewFromFlagSet(flag.CommandLine, os.Args[1:])
}

func NewFromFlagSet(f *flag.FlagSet, args []string) (Configuration, error) {
	var url, token, insecureStr string
	var insecure bool
	f.StringVar(&url, "argocd-url", "", "URL of the Argo CD server to query")
	f.StringVar(&token, "argocd-token", "", "The token to query Argo CD (will be expanded if specified as $ENV_VAR)")
	f.StringVar(&insecureStr, "insecure", "false", "Allow insecure TLS connections")
	if err := f.Parse(args); err != nil {
		return Configuration{}, err
	}
	if strings.HasPrefix(url, "$") {
		url = os.ExpandEnv(url)
	}
	if strings.HasPrefix(token, "$") {
		token = os.ExpandEnv(token)
	}
	if strings.HasPrefix(insecureStr, "$") {
		insecureStr = os.ExpandEnv(insecureStr)
	}
	insecure, err := strconv.ParseBool(insecureStr)
	if err != nil {
		return Configuration{}, err
	}
	return Configuration{
		URL:      url,
		Token:    token,
		Insecure: insecure,
	}, nil
}
