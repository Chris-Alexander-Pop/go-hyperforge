package cognito

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/cloud"
)

type Adapter struct {
	client   *cognitoidentityprovider.Client
	clientID string
}

func New(ctx context.Context, clientID string) (*Adapter, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		client:   cognitoidentityprovider.NewFromConfig(cfg),
		clientID: clientID,
	}, nil
}

func (a *Adapter) SignUp(ctx context.Context, username, password string, attributes map[string]string) error {
	input := &cognitoidentityprovider.SignUpInput{
		ClientId:       aws.String(a.clientID),
		Username:       aws.String(username),
		Password:       aws.String(password),
		UserAttributes: make([]types.AttributeType, 0, len(attributes)),
	}

	for k, v := range attributes {
		input.UserAttributes = append(input.UserAttributes, types.AttributeType{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	_, err := a.client.SignUp(ctx, input)
	return err
}

func (a *Adapter) SignIn(ctx context.Context, username, password string) (*cloud.AuthResult, error) {
	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(a.clientID),
		AuthParameters: map[string]string{
			"USERNAME": username,
			"PASSWORD": password,
		},
	}

	out, err := a.client.InitiateAuth(ctx, input)
	if err != nil {
		return nil, err
	}

	if out.AuthenticationResult == nil {
		return nil, nil // Challenge required or other flow
	}

	return &cloud.AuthResult{
		AccessToken:  aws.ToString(out.AuthenticationResult.AccessToken),
		RefreshToken: aws.ToString(out.AuthenticationResult.RefreshToken),
		IDToken:      aws.ToString(out.AuthenticationResult.IdToken),
		ExpiresIn:    int(out.AuthenticationResult.ExpiresIn),
	}, nil
}
