package keycloakclient

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

//go:generate options-gen -out-filename=client_options.gen.go -from-struct=Options
type Options struct {
	basePath     string `option:"mandatory" validate:"required"`
	realm        string `option:"mandatory" validate:"required"`
	clientID     string `option:"mandatory" validate:"required"`
	clientSecret string `option:"mandatory" validate:"required"`
	production   bool   `option:"mandatory"`
	debugMode    bool
}

// Client is a tiny client to the KeyCloak realm operations. UMA configuration:
// http://localhost:3010/realms/Bank/.well-known/uma2-configuration
type Client struct {
	realm        string
	clientID     string
	clientSecret string
	cli          *resty.Client
}

func New(opts Options) (*Client, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options keycloakclient: %v", err)
	}

	if opts.production && opts.debugMode {
		zap.L().Warn("Debug mode is enabled for Keycloak client in production environment. Review configuration settings.")
	}

	cli := resty.New()
	cli.SetDebug(opts.debugMode)
	cli.SetBaseURL(opts.basePath)

	return &Client{
		realm:        opts.realm,
		clientID:     opts.clientID,
		clientSecret: opts.clientSecret,
		cli:          cli,
	}, nil
}
