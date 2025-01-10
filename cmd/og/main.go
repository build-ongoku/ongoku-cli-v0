package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/gopi/json"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/mainutil"
	"github.com/teejays/gokutil/ogconfig"
	"github.com/teejays/gokutil/panics"

	"github.com/build-ongoku/ongoku-cli/pkg/subcmd/create"
	"github.com/build-ongoku/ongoku-cli/pkg/subcmd/deploy"
)

const _version = "0.1.1" // increment this for every release

// _buildTimeCompiledAtStr will be populated during compile time
var _buildTimeCompiledAtStr string
var _compiledAt time.Time

func main() {
	// Build context (and cancel it at the end). This lets us gracefully cancel any long running operations.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := mainHelper(ctx)
	if err != nil {
		log.Error(ctx, "Could not complete the request.", "error", err)
		os.Exit(1)
	}
}

type Args struct {
	mainutil.ParentArgs

	// Auth          *auth.Args   `arg:"subcommand:auth" help:"Authentication related commands"`
	Create *create.Args `arg:"subcommand:create" help:"Create a new Ongoku app."`
	Deploy *deploy.Args `arg:"subcommand:deploy" help:"Deployment related commands"`

	// Flags
	AppRootFromCurrDirPath string `arg:"-d,--app-dir" help:"The root directory of the Ongoku app. Defaults to current dircetory." default:"."`
}

func (v *Args) Version() string {
	return fmt.Sprintf("Ongoku CLI: Version %s\nBuildtime: %s\n", _version, _compiledAt)
}

func (v *Args) Parse(ctx context.Context) error {

	err := mainutil.ParseArgs(ctx, "Ongoku CLI", v)
	if err != nil {
		return err
	}

	return nil
}

func mainHelper(ctx context.Context) error {
	var err error

	if _buildTimeCompiledAtStr == "" {
		return errors.New("Build time for this binary is not set. This binary is not built correctly or may be corrupted.")
	}
	_compiledAt, err = time.Parse(time.RFC3339, _buildTimeCompiledAtStr)
	if err != nil {
		return errutil.Wrap(err, "Parsing the build time. This binary is not built correctly or may be corrupted.")
	}

	// Parse the command line args
	var args Args
	err = args.Parse(ctx)
	if err != nil {
		if errors.Is(err, mainutil.ErrCleanExit) {
			return nil
		}
		return errutil.Wrap(err, "Parsing command line args")
	}

	// Run the command
	err = run(ctx, &args)
	if err != nil {
		return errutil.Wrap(err, "Running command [og]")
	}

	return nil

}

func run(ctx context.Context, args *Args) error {
	panics.IfNil(args, "Args cannot be nil")

	var err error
	somethingDone := false

	// Set the log level specifically for this run
	log.Init(args.LogLevel)

	if args.Create != nil {

		// Create is a unique branch because 1) no config to start with, 2) app root dir path doesn't apply yet
		somethingDone = true
		args.Create.GokuVersion = _version

		log.Debug(ctx, "Running sub-command [create]", "args", json.MustPrettyPrint(args.Create))
		err = create.Run(ctx, args.Create)
		if err != nil {
			return errutil.Wrap(err, "Running sub-command [create]")
		}

	} else {

		// Initialize the config
		err = ogconfig.InitializeConfig("", &ogconfig.CLIConfig{
			AppRootFromCurrDirPath: args.AppRootFromCurrDirPath,
		})
		if err != nil {
			return errutil.Wrap(err, "Initializing config")
		}
		cfg := ogconfig.GetConfig()

		// if args.Auth != nil {
		// 	somethingDone = true

		// 	log.Debug(ctx, "Running sub-command [auth]", "args", json.MustPrettyPrint(args.Auth))
		// 	err = auth.Run(ctx, args.Auth)
		// 	if err != nil {
		// 		return errutil.Wrap(err, "Running sub-command [auth]")
		// 	}
		// }

		if args.Deploy != nil {
			somethingDone = true

			log.Debug(ctx, "Running sub-command [deploy]", "args", json.MustPrettyPrint(args.Deploy))
			err = deploy.Run(ctx, cfg, args.Deploy)
			if err != nil {
				return errutil.Wrap(err, "Running sub-command [deploy]")
			}
		}
	}

	if !somethingDone {
		return fmt.Errorf("Please provide a subcommand.")
	}

	return nil

}
