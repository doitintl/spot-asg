package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/doitintl/spot-asg/internal/aws/autoscaling"
	"github.com/doitintl/spot-asg/internal/aws/sts"
	"github.com/urfave/cli/v2"
)

var (
	// main context
	mainCtx context.Context
	// Version contains the current version.
	Version = "dev"
	// BuildDate contains a string with the build date.
	BuildDate = "unknown"
	// GitCommit git commit SHA
	GitCommit = "dirty"
	// GitBranch git branch
	GitBranch = "master"
	// app name
	appName = "spot-asg"
)

func getCallerIdentity(c *cli.Context) error {
	log.Printf("getting AWS caller identity with %s", c.FlagNames())
	checker := sts.NewRoleChecker(c.String("role-arn"), c.String("external-id"), c.String("region"))
	result, err := checker.CheckRole(mainCtx)
	if err != nil {
		return err
	}
	log.Print(result)
	return nil
}

func listAutoscalingGroups(c *cli.Context) error {
	tags := parseTags(c.StringSlice("tags"))
	log.Printf("get autoscaling groups with #{tags}")
	lister := autoscaling.NewAsgLister(c.String("role-arn"), c.String("external-id"), c.String("region"))
	result, err := lister.ListGroups(mainCtx, tags)
	if err != nil {
		return err
	}
	log.Print(result)
	return nil
}

func parseTags(list []string) map[string]string {
	tags := make(map[string]string, len(list))
	for _, t := range list {
		kv := strings.Split(t, "=")
		if len(kv) == 2 {
			tags[kv[0]] = kv[1]
		}
	}
	return tags
}

func init() {
	// handle termination signal
	mainCtx = handleSignals()
}

func handleSignals() context.Context {
	// Graceful shut-down on SIGINT/SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// create cancelable context
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		sid := <-sig
		log.Printf("received signal: %d\n", sid)
		log.Println("canceling main command ...")
	}()

	return ctx
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "bool",
				Value: true,
				Usage: "boolean app flag",
			},
			&cli.StringFlag{
				Name:  "role-arn",
				Usage: "role ARN to assume",
			},
			&cli.StringFlag{
				Name:  "external-id",
				Usage: "external ID to assume role with",
			},
			&cli.StringFlag{
				Name:    "region",
				Usage:   "the AWS Region to send the request to",
				EnvVars: []string{"AWS_DEFAULT_REGION"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "list-autoscaling-groups",
				Usage:  "list EC2 auto scaling groups, filtered with tags",
				Action: listAutoscalingGroups,
			},
			{
				Name:   "get-caller-identity",
				Usage:  "get AWS caller identity",
				Action: getCallerIdentity,
			},
		},
		Name:    appName,
		Usage:   "update/create MixedInstancePolicy for Amazon EC2 Auto Scaling groups",
		Version: Version,
	}
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("%s %s\n", appName, Version)
		fmt.Printf("  Build date: %s\n", BuildDate)
		fmt.Printf("  Git commit: %s\n", GitCommit)
		fmt.Printf("  Git branch: %s\n", GitBranch)
		fmt.Printf("  Built with: %s\n", runtime.Version())
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
