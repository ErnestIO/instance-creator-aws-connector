/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/nats-io/nats"
)

var nc *nats.Conn
var natsErr error

func eventHandler(m *nats.Msg) {
	var i Event

	err := i.Process(m.Data)
	if err != nil {
		return
	}

	if err = i.Validate(); err != nil {
		i.Error(err)
		return
	}

	err = createInstance(&i)
	if err != nil {
		i.Error(err)
		return
	}

	i.Complete()
}

func createInstance(ev *Event) error {
	creds := credentials.NewStaticCredentials(ev.DatacenterAccessKey, ev.DatacenterAccessToken, "")
	svc := ec2.New(session.New(), &aws.Config{
		Region:      aws.String(ev.DatacenterRegion),
		Credentials: creds,
	})

	req := ec2.RunInstancesInput{
		SubnetId:     aws.String(ev.NetworkAWSID),
		ImageId:      aws.String(ev.InstanceImage),
		InstanceType: aws.String(ev.InstanceType),
		KeyName:      aws.String(ev.InstanceKeyPair),
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
	}

	for _, sg := range ev.SecurityGroupAWSIDs {
		req.SecurityGroupIds = append(req.SecurityGroupIds, aws.String(sg))
	}

	resp, err := svc.RunInstances(&req)
	if err != nil {
		return err
	}

	ev.InstanceAWSID = *resp.Instances[0].InstanceId

	return nil
}

func main() {
	natsURI := os.Getenv("NATS_URI")
	if natsURI == "" {
		natsURI = nats.DefaultURL
	}

	nc, natsErr = nats.Connect(natsURI)
	if natsErr != nil {
		log.Fatal(natsErr)
	}

	fmt.Println("listening for instance.create.aws")
	nc.Subscribe("instance.create.aws", eventHandler)

	runtime.Goexit()
}
