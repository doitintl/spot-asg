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

	"github.com/doitintl/spot-asg/internal/aws/eventbridge"

	"github.com/doitintl/spot-asg/internal/aws/autoscaling"
	"github.com/doitintl/spot-asg/internal/aws/sts"
	"github.com/urfave/cli/v2"

	"github.com/aws/aws-lambda-go/lambda"
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

// handle Linux innteruption signals
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

func init() {
	// handle termination signal
	mainCtx = handleSignals()
}

func getCallerIdentity(role sts.AssumeRoleInRegion) error {
	checker := sts.NewRoleChecker(role)
	result, err := checker.CheckRole(mainCtx)
	if err != nil {
		return err
	}
	log.Print(result)
	return nil
}

func listAutoscalingGroups(asgRole, ebRole sts.AssumeRoleInRegion, eventBusArn string, tags map[string]string) error {
	lister := autoscaling.NewAsgLister(asgRole)
	result, err := lister.ListGroups(mainCtx, tags)
	if err != nil {
		return err
	}
	if eventBusArn != "" {
		publisher := eventbridge.NewAsgPublisher(ebRole, eventBusArn)
		err := publisher.PublishAsgGroups(mainCtx, result)
		if err != nil {
			return err
		}
	} else {
		log.Print(result)
	}
	return nil
}

//=========== CLI Commands ===========

func getCallerIdentityCmd(c *cli.Context) error {
	log.Printf("getting AWS caller identity with %s", c.FlagNames())
	return getCallerIdentity(
		sts.AssumeRoleInRegion{
			Arn:        c.String("role-arn"),
			ExternalID: c.String("external-id"),
			Region:     c.String("region"),
		})
}

func listAutoscalingGroupsCmd(c *cli.Context) error {
	tags := parseTags(c.StringSlice("tags"))
	log.Printf("get autoscaling groups with %v", tags)
	return listAutoscalingGroups(
		sts.AssumeRoleInRegion{
			Arn:        c.String("role-arn"),
			ExternalID: c.String("external-id"),
			Region:     c.String("region")},
		sts.AssumeRoleInRegion{
			Arn:        c.String("eb-role-arn"),
			ExternalID: c.String("eb-external-id"),
			Region:     c.String("eb-region"),
		},
		c.String("eb-eventbus-arn"),
		tags,
	)
}

func handleLabmdaCmd(c *cli.Context) error {
	lambda.StartWithContext(mainCtx, listAsgLambdaRequest)
	return nil
}

//=========== Lambda Handlers ===========

type scanRequest struct {
	asgRole     sts.AssumeRoleInRegion `json:"ags-role"`
	ebRole      sts.AssumeRoleInRegion `json:"eb-role"`
	eventBusArn string                 `json:"eb-eventbus-arn"`
	tags        map[string]string      `json:"tags"`
}

func listAsgLambdaRequest(ctx context.Context, req scanRequest) (string, error) {
	err := listAutoscalingGroups(req.asgRole, req.ebRole, req.eventBusArn, req.tags)
	if err != nil {
		return "error", err
	}
	return "done", nil
}

//=========== MAIN ===========

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:        "lambda",
				Description: "lambda mode",
				Action:      handleLabmdaCmd,
			},
			{
				Name:        "cli",
				Description: "command line mode",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "role-arn",
						Usage: "role ARN to assume",
					},
					&cli.StringFlag{
						Name:  "external-id",
						Usage: "external ID to assume role with",
					},
					&cli.StringFlag{
						Name:  "region",
						Usage: "the AWS Region to send the request to",
					},
				},
				Subcommands: []*cli.Command{

					{
						Name:   "list-autoscaling-groups",
						Usage:  "list EC2 auto scaling groups, filtered with tags",
						Action: listAutoscalingGroupsCmd,
						Flags: []cli.Flag{
							&cli.StringSliceFlag{
								Name:  "tags",
								Usage: "tags to filter by (syntax: key=value)",
							},
							&cli.StringFlag{
								Name:  "eb-eventbus-arn",
								Usage: "send list output to the specified Amazon EventBrige Event Bus",
							},
							&cli.StringFlag{
								Name:  "eb-role-arn",
								Usage: "role ARN to assume (for sending events to the Event Bus)",
							},
							&cli.StringFlag{
								Name:  "eb-external-id",
								Usage: "external ID to assume role with",
							},
							&cli.StringFlag{
								Name:  "eb-region",
								Usage: "the AWS Region of EventBridge Event Bus",
							},
						},
					},
					{
						Name:   "get-caller-identity",
						Usage:  "get AWS caller identity",
						Action: getCallerIdentityCmd,
					},
				},
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
