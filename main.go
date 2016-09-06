/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	ecc "github.com/ernestio/ernest-config-client"
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

func assignElasticIP(svc *ec2.EC2, instanceID string) (string, error) {
	// Create Elastic IP
	resp, err := svc.AllocateAddress(nil)
	if err != nil {
		return "", err
	}

	req := ec2.AssociateAddressInput{
		InstanceId:   aws.String(instanceID),
		AllocationId: resp.AllocationId,
	}
	_, err = svc.AssociateAddress(&req)
	if err != nil {
		return "", err
	}

	return *resp.PublicIp, nil
}

func createInstance(ev *Event) error {
	creds := credentials.NewStaticCredentials(ev.DatacenterAccessKey, ev.DatacenterAccessToken, "")
	svc := ec2.New(session.New(), &aws.Config{
		Region:      aws.String(ev.DatacenterRegion),
		Credentials: creds,
	})

	req := ec2.RunInstancesInput{
		SubnetId:         aws.String(ev.NetworkAWSID),
		ImageId:          aws.String(ev.InstanceImage),
		InstanceType:     aws.String(ev.InstanceType),
		PrivateIpAddress: aws.String(ev.InstanceIP),
		KeyName:          aws.String(ev.InstanceKeyPair),
		MaxCount:         aws.Int64(1),
		MinCount:         aws.Int64(1),
	}

	for _, sg := range ev.SecurityGroupAWSIDs {
		req.SecurityGroupIds = append(req.SecurityGroupIds, aws.String(sg))
	}

	resp, err := svc.RunInstances(&req)
	if err != nil {
		return err
	}

	builtInstance := ec2.DescribeInstancesInput{
		InstanceIds: []*string{resp.Instances[0].InstanceId},
	}

	err = svc.WaitUntilInstanceRunning(&builtInstance)
	if err != nil {
		return err
	}

	if ev.InstanceAssignElasticIP {
		ev.InstanceElasticIP, err = assignElasticIP(svc, *resp.Instances[0].InstanceId)
		if err != nil {
			return err
		}
	}

	ev.InstanceAWSID = *resp.Instances[0].InstanceId

	if resp.Instances[0].PublicIpAddress != nil {
		ev.InstancePublicIP = *resp.Instances[0].PublicIpAddress
	}

	return nil
}

func main() {
	nc = ecc.NewConfig(os.Getenv("NATS_URI")).Nats()

	fmt.Println("listening for instance.create.aws")
	nc.Subscribe("instance.create.aws", eventHandler)

	runtime.Goexit()
}
