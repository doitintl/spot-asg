[![](https://github.com/doitintl/spot-asg/workflows/docker/badge.svg)](https://github.com/doitintl/spot-asg/actions?query=workflow%3A"docker") [![Docker Pulls](https://img.shields.io/docker/pulls/doitintl/spot-asg.svg?style=popout)](https://hub.docker.com/r/doitintl/spot-asg) [![](https://images.microbadger.com/badges/image/doitintl/spot-asg.svg)](https://microbadger.com/images/doitintl/spot-asg "Get your own image badge on microbadger.com")

# spot-asg

The `spot-asg` can automatically uodate EC2 Auto Scaling group with Spot instances.

## Required AWS Permissions

The `spot-asg` can connect to the AWS API using default AWS credentials and can assume IAM Role. The IAM principle that runs the `spot-asg` binary/library must have permissions to assume the requested role (the same account; or cross-accout). 

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "sts:TagSession",
                "autoscaling:DescribeTags",
                "autoscaling:DescribeAutoScalingGroups"
            ],
            "Resource": "*"
        }
    ]
}
```


## Build

### Docker

The `spot-asg` uses Docker both as a CI tool and for releasing final `spot-asg` multi-architecture Docker image (`scratch` with updated `ca-credentials` package). The final Dockdr image pushed to the specified Docker registry (DockerHub by default) and to the GitHub Container Registry.

### Makefile

The `spot-asg` `Makefile` is used for task automation only: compile, lint, test and other.

### Continuous Integration

GitHub action `docker` is used for `spot-aqsg` CI.

#### Required GitHub secrets

Please specify the following GitHub secrets:

1. `DOCKER_USERNAME` - Docker Registry username
1. `DOCKER_PASSWORD` - Docker Registry password or token
1. `CR_PAT` - Current GitHub Personal Access Token
1. `DOCKER_REGISTRY` - _optional_; Docker Registry name, default to `docker.io`
1. `DOCKER_REPOSITORY` - _optional_; Docker image repository name, default to `$GITHUB_REPOSITORY` (i.e. `user/repo`)
