package auth

import (
	"context"
	"fmt"

	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/panics"

	"github.com/build-ongoku/ongoku-cli/pkg/client"
	"github.com/build-ongoku/ongoku-cli/pkg/local"
)

type Args struct {
	Login *LoginArgs `arg:"subcommand:login" help:"Login to the system"`
}

type LoginArgs struct {
	Username string `arg:"positional" help:"The username to login with"`
	Password string `arg:"positional" help:"The password to login with"`
}

func Run(ctx context.Context, args *Args) error {
	log.Debug(ctx, "Running sub-command", "command", "auth", "args", args)
	panics.If(args == nil, "Args cannot be nil")

	var err error
	somethingDone := false

	if args.Login != nil {
		somethingDone = true

		err = Login(ctx, args.Login)
		if err != nil {
			return errutil.Wrap(err, "Running sub-command [login]")
		}
	}

	if !somethingDone {
		return fmt.Errorf("No sub-command found")
	}

	return nil

}

func Login(ctx context.Context, args *LoginArgs) error {
	log.Debug(ctx, "Running sub-command", "command", "auth login", "args", args)
	panics.If(args == nil, "Args cannot be nil")

	if args.Username == "" {
		return fmt.Errorf("Username is empty")
	}
	if args.Password == "" {
		return fmt.Errorf("Password is empty")
	}

	// Create a new client, which will automatically make a request to the server to get a token
	client, err := client.NewClient(ctx, client.Creds{
		Email:    args.Username,
		Password: args.Password,
	})
	if err != nil {
		return errutil.Wrap(err, "Creating new client")
	}

	// Now that we are able to create a new client -- store the info in the config file so we don't have to login each time.
	// Save token in the config file
	cfg, err := local.LoadConfig(ctx, "")
	if err != nil {
		return errutil.Wrap(err, "Loading config")
	}
	cfg.Temporary.Token = client.Token
	err = local.SaveConfig(ctx, cfg)
	if err != nil {
		return errutil.Wrap(err, "Saving config")
	}

	return nil
}
