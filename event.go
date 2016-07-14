/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"encoding/json"
	"errors"
	"log"
)

var (
	ErrDatacenterIDInvalid          = errors.New("Datacenter VPC ID invalid")
	ErrDatacenterRegionInvalid      = errors.New("Datacenter Region invalid")
	ErrDatacenterCredentialsInvalid = errors.New("Datacenter credentials invalid")
	ErrNetworkInvalid               = errors.New("Network invalid")
	ErrInstanceNameInvalid          = errors.New("Instance name invalid")
	ErrInstanceImageInvalid         = errors.New("Instance image invalid")
	ErrInstanceTypeInvalid          = errors.New("Instance type invalid")
)

// Event stores the instance data
type Event struct {
	UUID                  string   `json:"_uuid"`
	BatchID               string   `json:"_batch_id"`
	ProviderType          string   `json:"_type"`
	DatacenterVPCID       string   `json:"datacenter_vpc_id"`
	DatacenterRegion      string   `json:"datacenter_region"`
	DatacenterAccessKey   string   `json:"datacenter_access_key"`
	DatacenterAccessToken string   `json:"datacenter_access_token"`
	NetworkAWSID          string   `json:"network_aws_id"`
	SecurityGroupAWSIDs   []string `json:"security_group_aws_ids"`
	InstanceAWSID         string   `json:"instance_aws_id,omitempty"`
	InstanceName          string   `json:"instance_name"`
	InstanceImage         string   `json:"instance_image"`
	InstanceType          string   `json:"instance_type"`
	InstanceKeyPair       string   `json:"instance_key_pair"`
	ErrorMessage          string   `json:"error,omitempty"`
}

// Validate checks if all criteria are met
func (ev *Event) Validate() error {
	if ev.DatacenterVPCID == "" {
		return ErrDatacenterIDInvalid
	}

	if ev.DatacenterRegion == "" {
		return ErrDatacenterRegionInvalid
	}

	if ev.DatacenterAccessKey == "" || ev.DatacenterAccessToken == "" {
		return ErrDatacenterCredentialsInvalid
	}

	if ev.NetworkAWSID == "" {
		return ErrNetworkInvalid
	}

	if ev.InstanceName == "" {
		return ErrInstanceNameInvalid
	}

	if ev.InstanceImage == "" {
		return ErrInstanceImageInvalid
	}

	if ev.InstanceType == "" {
		return ErrInstanceTypeInvalid
	}

	return nil
}

// Process the raw event
func (ev *Event) Process(data []byte) error {
	err := json.Unmarshal(data, &ev)
	if err != nil {
		nc.Publish("instance.create.aws.error", data)
	}
	return err
}

// Error the request
func (ev *Event) Error(err error) {
	log.Printf("Error: %s", err.Error())
	ev.ErrorMessage = err.Error()

	data, err := json.Marshal(ev)
	if err != nil {
		log.Panic(err)
	}
	nc.Publish("instance.create.aws.error", data)
}

// Complete the request
func (ev *Event) Complete() {
	data, err := json.Marshal(ev)
	if err != nil {
		ev.Error(err)
	}
	nc.Publish("instance.create.aws.done", data)
}
