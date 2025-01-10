package create

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/teejays/gokutil/cmdutil"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/naam"
	"github.com/teejays/gokutil/ogconfig"

	"github.com/build-ongoku/ongoku-cli/pkg/client/coreengine"
)

var llog = log.GetLogger().WithHeading("Goku Creator")

type Args struct {
	// Flags + Options
	Description    string   `arg:"--description" help:"Description of the app"`
	Components     []string `arg:"-c,--generate-components" help:"Components to turn on by default. Defaults to [backend,database,frontend]." `
	SkipGenerate   bool     `arg:"--skip-generate" default:"false" help:"Skip generating the code"`
	SkipGitInit    bool     `arg:"--skip-git-init" default:"false" help:"Skip initializing a git project"`
	SkipDevMigrate bool     `arg:"--skip-dev-migrate" default:"false" help:"Skip running initial migrations"`
	NoRollback     bool     `arg:"--no-rollback" default:"false" help:"Do not attempt to rollback if an error occurs"`

	// Positional Args
	AppName string `arg:"positional"`

	// Calculated
	appName     naam.Name
	appRootPath string

	// Internal
	GokuVersion string
}

func (a *Args) Validate(ctx context.Context) error {

	// Validate the app name
	if a.AppName == "" {
		return fmt.Errorf("Please provide an app name")
	}
	// Regex to ensure no special characters other than or -
	rgx, err := regexp.Compile(`^[a-zA-Z0-9-]*$`)
	if err != nil {
		return fmt.Errorf("Compiling regex: %w", err)
	}
	if !rgx.MatchString(a.AppName) {
		return fmt.Errorf("App name should only contain letters, numbers and -. How about naming it '%s'?", naam.New(a.AppName).ToKebab())
	}

	// Since we're just creating the app directory, we know the path of the app root relative to the current directory
	a.appName = naam.New(a.AppName)
	a.appRootPath = filepath.Join(".", a.AppName)

	// Validate the req + set any default values
	if len(a.Components) == 0 {
		log.Trace(ctx, "No components provided. Setting default components", "components", []string{"backend", "database", "frontend", "infra"})
		a.Components = []string{"backend", "database", "frontend", "infra"}
	} else if len(a.Components) == 1 {
		a.Components = strings.Split(a.Components[0], ",")
	}
	if a.Description == "" {
		log.Warn(ctx, "No description provided for the app. It is recommended that you add a description in the "+ogconfig.ProjectConfigFileName+" file.")
	}

	return nil

}

func RunWithInit(ctx context.Context, args *Args) error {

	return Run(ctx, args)
}

func Run(ctx context.Context, args *Args) error {
	var err error

	err = args.Validate(ctx)
	if err != nil {
		return errutil.Wrap(err, "Validating args")
	}

	// Get the default license
	cl, err := coreengine.NewClientFromDefaultLicenseFile(ctx)
	if err != nil {
		return errutil.Wrap(err, "Creating core engine client")
	}

	// Call the core engine to create the app
	cmdParts := []string{
		"create",
		args.appName.String(),
	}
	if args.Description != "" {
		cmdParts = append(cmdParts, "--description", args.Description)
	}
	if len(args.Components) > 0 {
		cmdParts = append(cmdParts, "--generate-components", strings.Join(args.Components, ","))
	}
	if args.SkipGenerate {
		cmdParts = append(cmdParts, "--skip-generate")
	}
	if args.SkipGitInit {
		cmdParts = append(cmdParts, "--skip-git-init")
	}
	if args.SkipDevMigrate {
		cmdParts = append(cmdParts, "--skip-dev-migrate")
	}
	if args.NoRollback {
		cmdParts = append(cmdParts, "--no-rollback")
	}
	cmdParts = append(cmdParts, "--log-level", log.GetLogLevel().String())

	cmd := exec.CommandContext(ctx, "goku", cmdParts...)

	err = cl.ExecuteCoreEngineCommand(ctx,
		cmd,
		cmdutil.ExecOptions{},
		true,
	)
	if err != nil {
		return errutil.Wrap(err, "Running core engine command")
	}

	return nil
}

// func Create(ctx context.Context, args *Args) error {
// 	llog.Info(ctx, "Creating app...", "args", json.MustPrettyPrint(args))

// 	var err error
// 	// Assume the args are validated

// 	var rollback bool
// 	var rollbackCmds []*exec.Cmd

// 	defer func() {
// 		if rollback && !args.NoRollback {
// 			llog.Info(ctx, "Attempting rollback...")
// 			for _, cmd := range rollbackCmds {
// 				llog.Debug(ctx, "Running...", "command", cmd.String())
// 				_, err = cmd.Output()
// 				if err != nil {
// 					llog.Info(ctx, "Could not rollback!", "Command", cmd, "error", err)
// 					break
// 				}
// 			}
// 		}
// 	}()

// 	// Step 1: Create the root directory
// 	err = cmdutil.ExecCmd(ctx, "mkdir", args.appRootPath)
// 	if err != nil {
// 		rollback = true
// 		return fmt.Errorf("Creating app directory: %w", err)
// 	}
// 	rollbackCmds = append(rollbackCmds, exec.CommandContext(ctx, "rm", "-rf", args.appRootPath))

// 	// Bare bones version of an App
// 	sparseApp := &builder.App{
// 		WithName:        utiltyp.WithName{Name: args.appName},
// 		WithDescription: utiltyp.WithDescription{Description: args.Description},
// 	}

// 	// Initialize bare bones Config (needed for component generation later)
// 	err = ogconfig.InitializeConfigCustomFileConfig(
// 		args.GokuVersion,
// 		&ogconfig.CLIConfig{
// 			AppRootFromCurrDirPath: args.appRootPath,
// 			CLIOrFileConfig: ogconfig.CLIOrFileConfig{
// 				Components: args.Components,
// 			},
// 		},
// 		ogconfig.FileConfig{
// 			AppName:      args.appName,
// 			GoModuleName: filepath.Join(args.appName.ToKebab(), "backend"),
// 		},
// 	)
// 	if err != nil {
// 		return errutil.Wrap(err, "Initializing config")
// 	}
// 	sparseCfg := ogconfig.GetConfig()

// 	// Step 2: Generate the project-config file
// 	err = component.GenerateComponent(ctx, projectconfig.GetComponent(), sparseApp, true)
// 	if err != nil {
// 		rollback = true
// 		return errutil.Wrap(err, "Generating project-config file")
// 	}

// 	// Once the file is generated, we can Initialize the config again
// 	err = ogconfig.InitializeConfig(args.GokuVersion, &sparseCfg.CLIConfig)
// 	if err != nil {
// 		rollback = true
// 		return errutil.Wrap(err, "Initializing config (2nd time)")
// 	}
// 	cfg := ogconfig.GetConfig()

// 	// Step 3: Boilterplate files
// 	log.Info(ctx, "Creating boilerplate files and directories...")
// 	err = HandleBoilerplate(ctx, cfg)
// 	if err != nil {
// 		rollback = true
// 		return errutil.Wrap(err, "Generating boilerplate files")
// 	}

// 	// Step 4: Generate the .env.* files
// 	// Hack: Make a fake dumy app since component generation relies on it.
// 	log.Info(ctx, "Creating .env.* files...")
// 	err = HandleEnvFiles(ctx, sparseApp, cfg)
// 	if err != nil {
// 		rollback = true
// 		return errutil.Wrap(err, "Generating .env.* files")
// 	}

// 	return nil
// }

// func HandleBoilerplate(ctx context.Context, cfg ogconfig.Config) error {
// 	var err error

// 	includePatterns := []string{
// 		"*",
// 	}
// 	excludePatterns := []string{
// 		"*/.DS_Store",
// 		"*/.next",
// 		"*/.next/*",
// 		"*/.yarn",
// 		"*/node_modules",
// 		"*/stub.embed",
// 		"stub.embed",
// 	}

// 	// Remove the frontend if not needed
// 	if !slices.Contains(cfg.Components, "frontend") {
// 		excludePatterns = append(excludePatterns, "apps")
// 	}
// 	// Remove the backend if not needed
// 	if !slices.Contains(cfg.Components, "backend") {
// 		excludePatterns = append(excludePatterns, "backend")
// 	}

// 	err = static.GenerateStatic(ctx, static.StaticInfo{
// 		EmbededPath: filepath.Join("root", "boilerplate"),
// 		OutputPath:  cfg.AppRootPath.FromCurrDir,
// 		SedSubstitutions: []cmdutil.FindSedDiff{
// 			{
// 				Old: "{{.goku_app_name}}",
// 				New: cfg.AppName.ToKebab(),
// 			},
// 			{
// 				Old: "{{.goku_app_backend_go_module_name}}",
// 				New: cfg.GoModuleName,
// 			},
// 		},
// 		Include: includePatterns,
// 		Exclude: excludePatterns,
// 	})
// 	if err != nil {
// 		return errutil.Wrap(err, "Generating boilerplate static")
// 	}

// 	return nil
// }

// func HandleEnvFiles(ctx context.Context, sparseApp *builder.App, cfg ogconfig.Config) error {
// 	var err error

// 	err = component.GenerateComponent(ctx, envs.GetComponent(), sparseApp, false)
// 	if err != nil {
// 		return errutil.Wrap(err, "Generating .env.* files")
// 	}

// 	// Copy the .env.* files from above to the root dir
// 	envFiles, err := filepath.Glob(filepath.Join(cfg.AppRootPath.FromCurrDir, ".goku", "generated", "envs", ".env.*"))
// 	if err != nil {
// 		return errutil.Wrap(err, "Finding .env.* files from [.goku/generated/env]")
// 	}
// 	if len(envFiles) == 0 {
// 		return fmt.Errorf("No .env.* files found from [.goku/generated/env]")
// 	}
// 	cpArgs := []string{}
// 	cpArgs = append(cpArgs, envFiles...)
// 	cpArgs = append(cpArgs, cfg.AppRootPath.FromCurrDir+"/.")
// 	err = cmdutil.ExecCmd(ctx, "cp", cpArgs...)
// 	if err != nil {
// 		return errutil.Wrap(err, "Copying .env.* files from [.goku/generated/env] to root dir")
// 	}

// 	return nil
// }
