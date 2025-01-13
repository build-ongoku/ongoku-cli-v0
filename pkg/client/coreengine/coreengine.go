package coreengine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/teejays/gokutil/cmdutil"
	"github.com/teejays/gokutil/env/envutil"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/panics"
)

// Default license key file path is $HOME/.ongoku/license
var _defaultLicenseFilePath string

func init() {
	homeDir := envutil.GetEnvVarStr("HOME")
	if homeDir == "" {
		panics.P("Cannot initialize default license file path. Env variable HOME is empty")
	}
	_defaultLicenseFilePath = filepath.Join(homeDir, ".ongoku", "license.txt")
}

type Client struct {
	license         string
	licenseFilePath string
}

func NewClientFromDefaultLicenseFile(ctx context.Context) (Client, error) {
	return NewClientFromLicenseFile(ctx, _defaultLicenseFilePath)
}

func NewClientFromLicenseFile(ctx context.Context, licenseFilePath string) (Client, error) {
	license, err := os.ReadFile(licenseFilePath)
	if err != nil {
		return Client{}, errutil.Wrap(err, "Reading license file")
	}
	c := Client{
		license:         string(license),
		licenseFilePath: licenseFilePath,
	}
	// Ensure that this works
	err = c.Validate(ctx)
	if err != nil {
		return Client{}, errutil.Wrap(err, "Ensuring that the core engine client is working")
	}
	return c, nil
}

func NewClientFromLicense(ctx context.Context, license []byte) (Client, error) {
	// Make a new client
	c := Client{
		license: string(license),
	}
	// Ensure that this works
	err := c.Validate(ctx)
	if err != nil {
		return Client{}, errutil.Wrap(err, "Ensuring that the core engine client is working")
	}
	return c, nil
}

func (c Client) Validate(ctx context.Context) error {
	// Ensure that this works
	cmd := exec.Command("goku", "version")
	err := c.ExecuteCoreEngineCommand(ctx, cmd, cmdutil.ExecOptions{}, true)
	if err != nil {
		return err
	}
	return nil
}

func (c Client) ExecuteCoreEngineCommand(ctx context.Context, cmd *exec.Cmd, opts cmdutil.ExecOptions, withLicense bool) error {

	// Todo: Decide whether to run it directly or through docker
	// For now, just run it directly

	// Add the license to the command
	if withLicense {
		if c.licenseFilePath != "" {
			cmd.Args = append(cmd.Args, "--license-file", c.licenseFilePath)
		} else if c.license != "" {
			cmd.Args = append(cmd.Args, "--license", c.license)
		} else {
			return errutil.New("No license set in the client")
		}
	}

	err := cmdutil.ExecOSCmdWithOpts(ctx, cmd, opts)
	if err != nil {
		return errutil.Wrap(err, "Running command")
	}

	return nil
}
