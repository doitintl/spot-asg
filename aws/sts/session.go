package sts

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

// MustAwsSession is a helper function that creates a new AWS Session and optional configuration
// This function is intended to be used to create AWS Client for any service
// for example,
//  sts.New(sts.MustAwsSession(roleARN, externalID, region))
func MustAwsSession(roleARN, externalID, region string) (*session.Session, *aws.Config) {
	// NewEC2Client constructs a new ec2 client with credentials and session
	sess := session.Must(session.NewSession())

	config := aws.NewConfig()

	if region != "" {
		config = config.WithRegion(region)
	}

	if (externalID != "") && (roleARN != "") {
		creds := stscreds.NewCredentials(sess, roleARN, func(p *stscreds.AssumeRoleProvider) {
			p.ExternalID = &externalID
		})

		config = config.WithCredentials(creds)
	}

	return sess, config
}
