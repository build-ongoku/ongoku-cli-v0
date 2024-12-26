package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	arg "github.com/alexflint/go-arg"

	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/gopi/json"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/ogconfig"
	"github.com/teejays/gokutil/panics"

	"github.com/build-ongoku/ongoku-cli/pkg/subcmd/auth"
	"github.com/build-ongoku/ongoku-cli/pkg/subcmd/deploy"
)

func main() {
	// Build context (and cancel it at the end). This lets us gracefully cancel any long running operations.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := mainHelper(ctx)
	if err != nil {
		log.Error(ctx, "Could not complete the request.", "error", err)
	}
}

type Args struct {
	Auth          *auth.Args   `arg:"subcommand:auth" help:"Authentication related commands"`
	Deploy        *deploy.Args `arg:"subcommand:deploy" help:"Deployment related commands"`
	VersionSubCmd *struct{}    `arg:"subcommand:version" help:"Print the version of the CLI. Also works with -v or --version."`
	HelpSubCmd    *struct{}    `arg:"subcommand:help" help:"Print the help message. Also works with -h or --help."`

	// Global Flags
	AppRootFromCurrDirPath string `arg:"-d,--app-dir" help:"The root directory of the Ongoku app (which is the directory where goku.yaml file is located. Defaults to current dircetory." default:"."`
}

func (v *Args) Version() string {
	return "Ongoku CLI Version 0.1.0\n"
}

var ErrCleanExit = errors.New("Clean exit")

func (v *Args) Parse(ctx context.Context) error {

	argParser, err := arg.NewParser(arg.Config{
		Program: "Ongoku CLI",
	}, v)
	if err != nil {
		return errutil.Wrap(err, "Creating new args parser")
	}

	log.Debug(ctx, "Parsing command line args")

	err = argParser.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, arg.ErrHelp) {
			argParser.WriteHelp(os.Stdout)
			return ErrCleanExit
		}
		if errors.Is(err, arg.ErrVersion) {
			fmt.Print(v.Version())
			return ErrCleanExit
		}
		return err
	}
	if v.VersionSubCmd != nil {
		fmt.Print(v.Version())
		return ErrCleanExit
	}
	if v.HelpSubCmd != nil {
		argParser.WriteHelp(os.Stdout)
		return ErrCleanExit
	}

	log.Debug(ctx, "Parsed args", "args", json.MustPrettyPrint(v))

	return nil
}

func mainHelper(ctx context.Context) error {
	var err error

	// Parse the command line args
	var args Args
	err = args.Parse(ctx)
	if err != nil {
		if errors.Is(err, ErrCleanExit) {
			return nil
		}
		return errutil.Wrap(err, "Parsing command line args")
	}

	// Do something with the args
	log.Debug(ctx, "Parsed command line args", "args", args)

	// Initialize the config
	err = ogconfig.InitializeConfig("", &ogconfig.CLIConfig{
		AppRootFromCurrDirPath: args.AppRootFromCurrDirPath,
	})
	if err != nil {
		return errutil.Wrap(err, "Initializing config")
	}
	cfg := ogconfig.GetConfig()

	err = run(ctx, cfg, &args)
	if err != nil {
		return errutil.Wrap(err, "Running command [og]")
	}

	return nil

}

func run(ctx context.Context, cfg ogconfig.Config, args *Args) error {
	panics.IfNil(args, "Args cannot be nil")

	var err error
	somethingDone := false

	if args.Auth != nil {
		somethingDone = true

		log.Debug(ctx, "Running sub-command [auth]", "args", json.MustPrettyPrint(args.Auth))
		err = auth.Run(ctx, args.Auth)
		if err != nil {
			return errutil.Wrap(err, "Running sub-command [auth]")
		}
	}

	if args.Deploy != nil {
		somethingDone = true

		log.Debug(ctx, "Running sub-command [deploy]", "args", json.MustPrettyPrint(args.Deploy))
		err = deploy.Run(ctx, cfg, args.Deploy)
		if err != nil {
			return errutil.Wrap(err, "Running sub-command [deploy]")
		}
	}

	if !somethingDone {
		return fmt.Errorf("Please provide a subcommand.")
	}

	return nil

}
