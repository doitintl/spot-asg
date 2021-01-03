[![docker](https://github.com/doitintl/spot-asg/workflows/docker/badge.svg)](https://github.com/doitintl/spot-asg/actions?query=workflow%3A"docker") [![Docker Pulls](https://img.shields.io/docker/pulls/doitintl/spot-asg.svg?style=popout)](https://hub.docker.com/r/doitintl/spot-asg) [![](https://images.microbadger.com/badges/image/doitintl/spot-asg.svg)](https://microbadger.com/images/doitintl/spot-asg "Get your own image badge on microbadger.com")

# spot-asg

The `spot-asg` can automatically uodate (or recommend an update) EC2 Auto Scaling groups in AWS Account with Spot instances.

```text
NAME:
   spot-asg - update/create MixedInstancePolicy for Amazon EC2 AutoScaling groups

USAGE:
   spot-asg [global options] command [command options] [arguments...]

VERSION:
   dev

COMMANDS:
   list                 list EC2 autoscaling groups, filtered by tags
   update               update EC2 autoscaling groups to maximize Spot usage
   recommend            recommend optimization for EC2 autoscaling groups to maximize Spot usage
   get-caller-identity  get AWS caller identity
   help, h              Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --lambda-mode        set to true if running as AWS Lambda (default: false)
   --role-arn value     role ARN to assume
   --external-id value  external ID to assume role with
   --region value       the AWS Region to send the request to
   --help, -h           show help (default: false)
   --version, -v        print the version (default: false)
```
## update command

```text
NAME:
spot-asg update - update EC2 autoscaling groups to maximize Spot usage

USAGE:
main update [command options] [arguments...]

OPTIONS:
--ignore-family                                                 ignore instance type family (default: false)
--ignore-generation                                             ignore instance type generation (default: false)
--multiply-factor-upper value, --mfu value                      apply multiply factor to define upper VCPU limit (default: 2)
--multiply-factor-lower value, --mfl value                      apply multiply factor to define lower VCPU limit (default: 2)
--ondemand-base-capacity value, --obc value                     capacity to be fulfilled by on-demand instances (VCPU weight) (default: 0)
--ondemand-percentage-above-base-capacity value, --opabc value  percentage of on-demand instances above base capacity (default: 0)
--tags value                                                    tags to filter by (syntax: key=value)
--help, -h                                                      show help (default: false)
```

## recommend command

```text
NAME:
   spot-asg recommend - recommend optimization for EC2 autoscaling groups to maximize Spot usage

USAGE:
   spot-asg recommend [command options] [arguments...]

OPTIONS:
   --eb-eventbus-arn value                                         send list output to the specified Amazon EventBrige Event Bus
   --eb-role-arn value                                             role ARN to assume for sending events to the Event Bus
   --eb-external-id value                                          external ID to assume role with
   --eb-region value                                               the AWS Region of EventBridge Event Bus
   --ignore-family                                                 ignore instance type family (default: false)
   --ignore-generation                                             ignore instance type generation (default: false)
   --multiply-factor-upper value, --mfu value                      apply multiply factor to define upper VCPU limit (default: 2)
   --multiply-factor-lower value, --mfl value                      apply multiply factor to define lower VCPU limit (default: 2)
   --ondemand-base-capacity value, --obc value                     capacity to be fulfilled by on-demand instances (VCPU weight) (default: 0)
   --ondemand-percentage-above-base-capacity value, --opabc value  percentage of on-demand instances above base capacity (default: 0)
   --tags value                                                    tags to filter by (syntax: key=value)
   --help, -h                                                      show help (default: false)
```

## Required AWS Permissions

The `spot-asg` can connect to the AWS API using default AWS credentials and can assume IAM Role. The IAM principle that runs the `spot-asg` binary/library must have permissions to assume the requested role (the same account; or cross-accout). 

The following IAM permissions are required to descover ASG on the target AWS Account:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "sts:TagSession",
                "autoscaling:DescribeTags",
                "autoscaling:DescribeAutoScalingGroups",
                "ec2:DescribeLaunchTemplateVersions"
            ],
            "Resource": "*"
        }
    ]
}
```

The `spot-asg` can send discovered ASG groups to Amazon EventBrige Event Bus, using default AWS credentials or assuming cross-account IAM Role. 

The following IAM Permissions are required to send results to the Event Bus
 
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "events:PutEvents",
            "Resource": "arn:aws:events:AWS_REGION:012345678901:event-bus/EVENT_BUS_NAME"
        }
    ]
}
```

## Build

### Docker

The `spot-asg` uses Docker both as a CI tool and for releasing final `spot-asg` multi-architecture Docker image (`scratch` with updated `ca-credentials` package). The final Docker image pushed to the specified Docker registry (DockerHub by default) and to the GitHub Container Registry.

#### Building with Docker

Use Docker BuildKit to build the image: either `buildx` command/plugin or set `DOCKER_BUILDKIT=1` environment variable.

### Makefile

The `spot-asg` `Makefile` is used for task automation only: compile, lint, test and other.

### Continuous Integration

GitHub action `docker` is used for `spot-asg` CI.

#### Required GitHub secrets

Please specify the following GitHub secrets:

1. `DOCKER_USERNAME` - Docker Registry username/owner
1. `DOCKER_PASSWORD` - Docker Registry password/token
1. `CR_PAT` - Current GitHub Personal Access Token (with `write/read` packages permission)
1. `DOCKER_REGISTRY` - _optional_; Docker Registry name, default to `docker.io`
1. `DOCKER_REPOSITORY` - _optional_; Docker image repository name, default to `$GITHUB_REPOSITORY` (i.e. `user/repo`)
