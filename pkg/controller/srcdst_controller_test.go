package controller

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
)

type mockEC2Client struct {
	ec2iface.EC2API
	res           *ec2.ModifyInstanceAttributeOutput
	err           error
	CalledCounter int
}

func (c *mockEC2Client) ModifyInstanceAttribute(*ec2.ModifyInstanceAttributeInput) (*ec2.ModifyInstanceAttributeOutput, error) {
	c.CalledCounter = c.CalledCounter + 1
	return c.res, c.err
}

func NewMockEC2Client() *mockEC2Client {
	return &mockEC2Client{CalledCounter: 0}
}

func TestDisableSrcDstIfEnabled(t *testing.T) {
	annotations := map[string]string{SrcDstCheckDisabledAnnotation: "true"}
	spec := v1.NodeSpec{ProviderID: "aws:///us-mock-1/i-abcdefgh"}
	node1 := &v1.Node{Spec: spec}
	node1.Annotations = annotations
	var tests = []struct {
		node                     *v1.Node
		disableSrcDstCheckCalled bool
	}{
		{&v1.Node{Spec: spec}, true},
		{node1, false},
	}

	ec2Client := NewMockEC2Client()

	c := &Controller{
		ec2Client: ec2Client,
		client:    fake.NewSimpleClientset(),
	}

	for _, tt := range tests {
		calledCount := ec2Client.CalledCounter
		c.disableSrcDstIfEnabled(tt.node)
		called := (ec2Client.CalledCounter - calledCount) > 0
		assert.Equal(
			t,
			called,
			tt.disableSrcDstCheckCalled,
			"Verify that ModifyInstanceAttribute will get called if node needs srcdstcheck disabled",
		)
		// TODO: validate that node did get updated with the annotation after ModifyInstanceAttribute is called
	}
}

func TestGetInstanceIDFromProviderID(t *testing.T) {

	var tests = []struct {
		providerID         string
		expectedInstanceID string
		expectedError      bool
	}{
		{"aws:///us-west-2a/i-09fc5a0ae524b0333", "i-09fc5a0ae524b0333", false},
		{"aws://us-west-2a/i-a123hd52", "i-a123hd52", false},
		{"gce://us-west-1a/test", "", true},
		{"this_will_fail", "", true},
		{"i-a123hd52", "", true},
	}

	for _, tt := range tests {
		instanceID, err := GetInstanceIDFromProviderID(tt.providerID)
		if !tt.expectedError {
			assert.Equal(
				t,
				tt.expectedInstanceID,
				*instanceID,
				"Check if instance ID is parsed out correctly from provider ID",
			)
		} else {
			assert.NotNil(
				t,
				err,
				err.Error(),
			)
		}
	}

}
