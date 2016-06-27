package cmd

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/dnephin/dobi/config"
	"github.com/dnephin/dobi/tasks"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/spf13/cobra"
)

const (
	// DefaultDockerAPIVersion is the default version of the docker API to use
	DefaultDockerAPIVersion = "1.23"
)

type dobiOptions struct {
	filename string
	verbose  bool
	quiet    bool
	tasks    []string
}

// NewRootCommand returns a new root command
func NewRootCommand() *cobra.Command {
	var opts dobiOptions

	cmd := &cobra.Command{
		Use:                   "dobi [flags] TASK [TASK...]",
		Short:                 "A build tool for Docker application.",
		SilenceUsage:          true,
		SilenceErrors:         true,
		TraverseChildCommands: true,
		Args: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.tasks = args
			return runDobi(opts)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			initLogging(opts.verbose, opts.quiet)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.filename, "filename", "f", "dobi.yaml", "Path to config file")
	flags.BoolVarP(&opts.verbose, "verbose", "v", false, "Verbose")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Quiet")

	flags.SetInterspersed(false)
	cmd.AddCommand(newListCommand(&opts))
	return cmd
}

func runDobi(opts dobiOptions) error {
	conf, err := config.Load(opts.filename)
	if err != nil {
		return err
	}

	client, err := buildClient()
	if err != nil {
		return fmt.Errorf("Failed to create client: %s", err)
	}

	return tasks.Run(tasks.RunOptions{Client: client, Config: conf, Tasks: opts.tasks})
}

func initLogging(verbose, quiet bool) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	if quiet {
		log.SetLevel(log.WarnLevel)
	}
	log.SetOutput(os.Stderr)
}

func buildClient() (*docker.Client, error) {
	apiVersion := os.Getenv("DOCKER_API_VERSION")
	if apiVersion == "" {
		apiVersion = DefaultDockerAPIVersion
	}
	// TODO: args for client
	client, err := docker.NewVersionedClientFromEnv(apiVersion)
	if err != nil {
		return nil, err
	}
	log.Info("Docker client created")
	return client, nil
}
