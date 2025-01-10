package deploy

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/teejays/gokutil/cmdutil"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/gopi/json"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/ogconfig"
)

type Args struct {
	DockerImage *DockerImageArgs `arg:"subcommand:docker-image" help:"Build and push docker images for the app, that can be deployed to the cloud."`
	K8sApply    *K8sApplyArgs    `arg:"subcommand:k8s-apply" help:"Apply the k8s deployment file(s) for the app."`
	Destroy     *struct{}        `arg:"subcommand:destroy" help:"Destroy the deployment in the cloud. This will delete the app from the cloud."`

	CommonFlags

	ogconfig.CLIConfig
	GokuVersion string
}

type CommonFlags struct {
	DeployIdentifier string `arg:"--deploy-identifier,env:GOKU_DEPLOY_IDENTIFIER" help:"The identifier to use for the deployment. This is used to identify the deployment in the cloud."`
}

type (
	DockerImageArgs struct {
		DockerImageFlags
	}

	DockerImageFlags struct {
		ImageRepo string `arg:"--image-repo,env:GOKU_DEPLOY_IMAGE_REPO" help:"The repo to use for the built images. If not provided, a default repo will be used."`
		ImageTag  string `arg:"--image-tag,env:GOKU_DEPLOY_IMAGE_TAG" help:"The tag to use for the built images. If not provided, a default tag will be used."`
		NoPush    bool   `arg:"--no-push" help:"Do not push the built images to the registry"`
	}
)

type (
	K8sApplyArgs struct {
		K8sApplyFlags
	}
	K8sApplyFlags struct{}
)

func RunWithInit(ctx context.Context, args *Args) error {

	// Load the env file(s), since they are needed to connect to the database

	// Load the goku.yaml file.
	// This is needed to get the app name
	err := ogconfig.InitializeConfig(args.GokuVersion, &args.CLIConfig)
	if err != nil {
		return fmt.Errorf("Initializing config: %w", err)
	}

	cfg := ogconfig.GetConfig()

	return Run(ctx, cfg, args)
}

func Run(ctx context.Context, cfg ogconfig.Config, args *Args) error {

	var somethingDone bool

	if args.CommonFlags.DeployIdentifier == "" {
		args.CommonFlags.DeployIdentifier = cfg.AppName.ToCompact()
		log.Warn(ctx, "DeployIdentifier not provided. Using default value.", "default", args.CommonFlags.DeployIdentifier)
	}

	// DockerImage
	if args.DockerImage != nil {
		somethingDone = true

		log.Info(ctx, "Running subcommand [docker-imahge]", "args", json.MustPrettyPrint(args.DockerImage))
		err := RunDockerImage(ctx, cfg, args.DockerImage, args.CommonFlags)
		if err != nil {
			return err
		}
	}

	// K8sApply
	if args.K8sApply != nil {
		somethingDone = true

		log.Info(ctx, "Running subcommand [k8s-apply]", "args", json.MustPrettyPrint(args.K8sApply))
		err := RunK8sApply(ctx, cfg, args.K8sApply, args.CommonFlags)
		if err != nil {
			return errutil.Wrap(err, "Running subcommand [k8s-apply]")
		}

	}

	// Destroy is not a part of all
	if args.Destroy != nil {
		somethingDone = true

		log.Info(ctx, "Running subcommand [destroy]")
		err := RunDestroy(ctx, cfg, args.Destroy, args.CommonFlags)
		if err != nil {
			return errutil.Wrap(err, "Running subcommand [destroy]")
		}
	}

	if !somethingDone {
		return fmt.Errorf("Please provide a subcommand.")
	}

	return nil
}

// RunDockerImage is a builds and pushes the docker image for the app to the registry.
func RunDockerImage(ctx context.Context, cfg ogconfig.Config, args *DockerImageArgs, commonFlags CommonFlags) error {
	var err error

	if commonFlags.DeployIdentifier == "" {
		commonFlags.DeployIdentifier = cfg.AppName.ToCompact()
		log.Warn(ctx, "DeployIdentifier not provided. Using default value.", "default", commonFlags.DeployIdentifier)
	}

	if args.ImageRepo == "" {
		args.ImageRepo = "iamteejay/goku"
		log.Warn(ctx, "ImageRepo not provided. Using default value.", "default", args.ImageRepo)
	}

	if args.ImageTag == "" {
		args.ImageTag = fmt.Sprintf("og-img-%s", commonFlags.DeployIdentifier)
		log.Warn(ctx, "ImageTag not provided. Using default value.", "default", args.ImageTag)
	}

	// Build the base image (which should have all the necessary files to run anything)
	log.Info(ctx, "DockerImage Step [1/	] Building & pushing deploy image...")
	err = cmdutil.DockerImageBuild(ctx,
		cmdutil.DockerBuildReq{
			DockerfilePath:  filepath.Join(cfg.AppRootPath.Full, "infra", ".goku", "static", "app.Dockerfile"),
			DockerImageRepo: args.ImageRepo,
			DockerImageTag:  args.ImageTag,
			NoPush:          args.NoPush,
			Platforms:       []string{"linux/amd64"},
		},
		cmdutil.ExecOptions{
			Dir: cfg.AppRootPath.Full,
		},
	)
	if err != nil {
		return errutil.Wrap(err, "Building docker image [%s] using file [%s]", args.ImageTag, "app.Dockerfile")
	}

	return nil

}

func RunK8sApply(ctx context.Context, cfg ogconfig.Config, args *K8sApplyArgs, commonFlags CommonFlags) error {

	var err error

	// Digital Ocean App Platform
	/*
		// doctl app create --upsert --spec .goku/generated/do/app.yaml
		err := cmdutil.ExecCmdFromDir(ctx, cfg.AppRootPath.Full,
			"doctl", "app",
			"create", "--upsert",
			"--spec", filepath.Join(".goku", "generated", "do", "app.yaml"),
		)
		if err != nil {
			return errutil.Wrap(err, "Running command [doctl app create]")
		}
	*/

	// Kubernetes (kubectl should be setup where this command is run)
	err = cmdutil.ExecCmdFromDir(ctx, cfg.AppRootPath.Full,
		"kubectl", "apply", "-f", filepath.Join("infra", ".goku", "generated", "k3s", "app.yaml"),
	)
	if err != nil {
		return errutil.Wrap(err, "Running command [kubectl apply]")
	}

	// Todo: Add a wait here to wait for the pods to be ready
	// kubectl wait -n ongoku --for=condition=Ready pod -l identifier=ongoku --timeout=5m
	{
		cmd := exec.CommandContext(ctx, "kubectl", "wait", "-n", "ongoku", "--for=condition=Ready", "pod", "-l", fmt.Sprint("identifier=", commonFlags.DeployIdentifier), "--timeout=5m")
		err = cmdutil.ExecOSCmdWithOpts(ctx, cmd, cmdutil.ExecOptions{
			Dir: cfg.AppRootPath.Full,
		})
		if err != nil {
			return errutil.Wrap(err, "Waiting for pods to be ready")
		}
	}

	return nil
}

func RunDestroy(ctx context.Context, cfg ogconfig.Config, args *struct{}, commondFlags CommonFlags) error {

	// Kubernetes (kubectl should be setup where this command is run)
	err := cmdutil.ExecCmdFromDir(ctx, cfg.AppRootPath.Full,
		"kubectl", "delete", "-f", filepath.Join("infra", ".goku", "generated", "k3s", "app.yaml"),
	)
	if err != nil {
		return errutil.Wrap(err, "Running command [kubectl delete]")
	}

	return nil
}
