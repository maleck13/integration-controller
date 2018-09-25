package fuse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/sirupsen/logrus"

	errors2 "k8s.io/apimachinery/pkg/api/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	"github.com/integr8ly/integration-controller/pkg/integration"
	"github.com/pkg/errors"
)

const (
	connectionIDKey = "connectionID"
	realmKey        = "realm"
	msgHostKey      = "msgHost"
)

var validIntegrationTypes = []string{"amqp", "api"}

type Integrator struct {
	enmasseService integration.EnMasseService
	xsrfToken      string
	saToken        string
	saUser         string
	k8sclient      kubernetes.Interface
	namespace      string
	httpClient     *http.Client
}

func NewIntegrator(service integration.EnMasseService, k8sClient kubernetes.Interface, httpClient *http.Client, namespace, saToken, saUser string) *Integrator {
	return &Integrator{enmasseService: service, k8sclient: k8sClient, xsrfToken: "awesome",
		saToken: saToken, saUser: saUser, namespace: namespace, httpClient: httpClient}
}

// Integrate
func (f *Integrator) Integrate(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	logrus.Debug("handling fuse integration ", integration.Spec.IntegrationType)
	switch integration.Spec.IntegrationType {
	case "amqp":
		intCp := integration.DeepCopy()
		return f.addAMQPIntegration(ctx, intCp)
	}
	return nil, nil
}

func (f *Integrator) DisIntegrate(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	logrus.Debug("handling fuse removing integration ", integration.Spec.IntegrationType)
	switch integration.Spec.IntegrationType {
	case "amqp":
		return f.removeAMQPIntegration(ctx, integration)
	}
	return nil, nil
}

func (f *Integrator) Integrates() string {
	return v1alpha1.FuseIntegrationTarget
}

func (f *Integrator) Validate(i *v1alpha1.Integration) error {
	valid := false
	var err error
	for _, it := range validIntegrationTypes {
		if i.Spec.IntegrationType == it {
			valid = true
		}
	}
	if valid != true {
		err = errors.New("unknown integration type should be one of " + strings.Join(validIntegrationTypes, ","))
		return err
	}
	if _, ok := i.Status.IntegrationMetaData[msgHostKey]; !ok && i.Spec.IntegrationType == "amqp" {
		return errors.New("expected to find the key " + msgHostKey + " in the metadata but it was missing")
	}
	if _, ok := i.Status.IntegrationMetaData[realmKey]; !ok && i.Spec.IntegrationType == "amqp" {
		return errors.New("expected to find the key " + realmKey + " in the metadata but it was missing")
	}
	return nil
}

func (f *Integrator) removeAMQPIntegration(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	ic := integration.DeepCopy()
	realm := ic.Status.IntegrationMetaData[realmKey]
	if err := f.enmasseService.DeleteUser(integration.Name, realm); err != nil {
		return nil, err
	}
	if err := f.deleteConnection("amqp", ic.Status.IntegrationMetaData[connectionIDKey]); err != nil {
		return nil, err
	}
	return ic, nil
}

func (f *Integrator) addAMQPIntegration(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	ic := integration.DeepCopy()
	realm := ic.Status.IntegrationMetaData[realmKey]
	if realm == "" {
		return nil, errors.New("missing realm in metadata")
	}
	logrus.Debug("integration metaData ", realm)
	//need to avoid creating users so need determinstic user name
	u, err := f.enmasseService.CreateUser(integration.Name, realm)
	if err != nil && !errors2.IsAlreadyExists(err) {
		return nil, errors.Wrap(err, "integration failed. Could not generate new user in enmasse keycloak")
	}
	amqpHost := fmt.Sprintf("amqp://%s?amqp.saslMechanisms=PLAIN", integration.Status.IntegrationMetaData[msgHostKey])
	id, err := f.createConnection("amqp", integration.Name, u.UserName, u.Password, amqpHost)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create amqp connection in fuse")
	}
	ic.Status.Phase = v1alpha1.PhaseComplete
	if id != "" {
		ic.Status.IntegrationMetaData[connectionIDKey] = id
	}
	return ic, nil
}

func (f *Integrator) deleteConnection(connectionType, connectionID string) error {
	host := "syndesis-server." + f.namespace + ".svc"
	logrus.Info("Deleting connection")
	apiHost := "http://" + host + "/api/v1/connections/" + connectionID
	req, err := http.NewRequest("DELETE", apiHost, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-FORWARDED-USER", f.saUser)
	req.Header.Set("X-FORWARDED-ACCESS-TOKEN", f.saToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("SYNDESIS-XSRF-TOKEN", f.xsrfToken)
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return errors.New("unexpected response from fuse api status code " + resp.Status)
	}
	return nil
}

func (f *Integrator) createConnection(connectionType, connectionName, username, password, url string) (string, error) {
	host := "syndesis-server." + f.namespace + ".svc"
	logrus.Info("Creating connection")
	apiHost := "http://" + host + "/api/v1/connections"
	var body *bytes.Buffer
	var err error
	switch connectionType {
	case "amqp":
		logrus.Infof("creating new AMQP connection: %s", connectionName)
		body, err = newAMQPConnectionCreatePostBody(username, password, url, connectionName)
		if err != nil {
			return "", err
		}
	case "http":
		logrus.Infof("creating new HTTP connection: %s", connectionName)
		body, err = newHTTPConnectionCreatePostBody(url, connectionName)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unknown connection type '%s'", connectionType)
	}

	req, err := http.NewRequest("POST", apiHost, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-FORWARDED-USER", f.saUser)
	req.Header.Set("X-FORWARDED-ACCESS-TOKEN", f.saToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("SYNDESIS-XSRF-TOKEN", f.xsrfToken)
	rsp, err := f.httpClient.Do(req)
	if err != nil {
		rspBodyBytes, _ := ioutil.ReadAll(rsp.Body)
		logrus.Infof("response to bad create request:: %s", string(rspBodyBytes))
		return "", err
	}

	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		logrus.Infof("request headers: %+v", req.Header)
		logrus.Info("response status from fuse ", rsp.Status)
		bodyBytes, _ := ioutil.ReadAll(body)
		logrus.Infof("request body: %s", string(bodyBytes))
		rspBodyBytes, _ := ioutil.ReadAll(rsp.Body)
		logrus.Infof("response to bad create request:: %s", string(rspBodyBytes))
		return "", fmt.Errorf("error creating connection")
	}

	dec := json.NewDecoder(rsp.Body)
	connResp := &connectionCreationResponse{}
	if err := dec.Decode(connResp); err != nil {
		return "", errors.Wrap(err, " failed to decode connection creation response")
	}
	return connResp.ID, nil

}

func newHTTPConnectionCreatePostBody(url, name string) (*bytes.Buffer, error) {
	body := connectionCreatePost{
		Connector: connector{
			Description:     "Invoke various HTTP methods.",
			Icon:            "http",
			ComponentScheme: "http4",
			ActionsSummary: connectorActionsSummary{
				TotalActions: 2,
			},
			Uses:    0,
			ID:      "http4",
			Version: 8,
			Actions: []connectorAction{
				{
					ID:          "io.syndesis.connector:connector-http:http4-invoke-url",
					Name:        "Invoke URL",
					Description: "Invoke an http endpoint URL",
					Descriptor: connectorActionDescriptor{
						ConnectorFactory: "io.syndesis.connector.http.HttpConnectorFactories$Http4",
						InputDataShape: connectorDataShape{
							Kind: "any",
						},
						OutputDataShape: connectorDataShape{
							Kind: "none",
						},
						PropertyDefinitionSteps: []connectorStep{
							{
								Description: "properties",
								Name:        "properties",
								Properties: map[string]connectorProperty{
									"httpMethod": connectorProperty{
										DefaultValue: "GET",
										Deprecated:   false,
										LabelHint:    "The specific http method to execute.",
										DisplayName:  "Http Method",
										Group:        "common",
										JavaType:     "java.lang.String",
										Kind:         "parameter",
										Required:     false,
										Secret:       false,
										Type:         "string",
										Enum: []connectorEnum{
											{
												Label: "GET",
												Value: "GET",
											},
											{
												Label: "PUT",
												Value: "PUT",
											},
											{
												Label: "POST",
												Value: "POST",
											},
											{
												Label: "DELETE",
												Value: "DELETE",
											},
											{
												Label: "HEAD",
												Value: "HEAD",
											},
											{
												Label: "OPTIONS",
												Value: "OPTIONS",
											},
											{
												Label: "TRACE",
												Value: "TRACE",
											},
											{
												Label: "PATCH",
												Value: "PATCH",
											},
										},
									},
									"path": connectorProperty{
										Deprecated:  false,
										LabelHint:   "Endpoint Path (eg '/path/to/endpoint')",
										DisplayName: "URL Path",
										Group:       "common",
										JavaType:    "java.lang.String",
										Kind:        "parameter",
										Required:    false,
										Secret:      false,
										Type:        "string",
									},
								},
							},
						},
					},
					ActionType: "connector",
					Pattern:    "To",
				},
				{
					ID:          "io.syndesis.connector:connector-http:http4-periodic-invoke-url",
					Name:        "Periodic invoke URL",
					Description: "Periodically invoke an http endpoint URL",
					Descriptor: connectorActionDescriptor{
						ConnectorFactory: "io.syndesis.connector.http.HttpConnectorFactories$Http4",
						InputDataShape: connectorDataShape{
							Kind: "none",
						},
						OutputDataShape: connectorDataShape{
							Kind: "any",
						},
						PropertyDefinitionSteps: []connectorStep{
							{
								Description: "properties",
								Name:        "properties",
								Properties: map[string]connectorProperty{
									"httpMethod": connectorProperty{
										DefaultValue: "GET",
										Deprecated:   false,
										LabelHint:    "The specific http method to execute.",
										DisplayName:  "Http Method",
										Group:        "common",
										JavaType:     "java.lang.String",
										Kind:         "parameter",
										Required:     false,
										Secret:       false,
										Type:         "string",
										Enum: []connectorEnum{
											{
												Label: "GET",
												Value: "GET",
											},
											{
												Label: "PUT",
												Value: "PUT",
											},
											{
												Label: "POST",
												Value: "POST",
											},
											{
												Label: "DELETE",
												Value: "DELETE",
											},
											{
												Label: "HEAD",
												Value: "HEAD",
											},
											{
												Label: "OPTIONS",
												Value: "OPTIONS",
											},
											{
												Label: "TRACE",
												Value: "TRACE",
											},
											{
												Label: "PATCH",
												Value: "PATCH",
											},
										},
									},
									"path": connectorProperty{
										Deprecated:  false,
										LabelHint:   "Endpoint Path",
										PlaceHolder: "eg '/path/to/endpoint'",
										DisplayName: "URL Path",
										Group:       "common",
										JavaType:    "java.lang.String",
										Kind:        "parameter",
										Required:    false,
										Secret:      false,
										Type:        "string",
									},
									"schedulerExpression": connectorProperty{
										DefaultValue: "1000",
										Deprecated:   false,
										LabelHint:    "Delay in milliseconds between scheduling (executing).",
										DisplayName:  "Period",
										Group:        "consumer",
										JavaType:     "long",
										Kind:         "parameter",
										Required:     false,
										Secret:       false,
										Type:         "duration",
									},
								},
							},
						},
					},
					ActionType: "connector",
					Pattern:    "From",
				},
			},
			Tags: []string{
				"verifier",
			},
			Name: "HTTP",
			Properties: map[string]connectorProperty{
				"baseUrl": connectorProperty{
					Deprecated:  false,
					LabelHint:   "Base Http Endpoint URL",
					PlaceHolder: "eg 'www.redhat.com'",
					DisplayName: "Base URL",
					Group:       "common",
					JavaType:    "java.lang.String",
					Kind:        "parameter",
					Required:    true,
					Secret:      false,
					Type:        "string",
				},
			},
			Dependencies: []connectorDependency{
				{
					Type: "MAVEN",
					ID:   "io.syndesis.connector:connector-http:1.5-SNAPSHOT",
				},
			},
		},
		Icon:        "http",
		ConnectorID: "http4",
		ConfiguredProperties: configuredProperties{
			BaseURL: url,
		},
		Name:        name,
		Description: "",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(bodyBytes), nil
}

func newAMQPConnectionCreatePostBody(username, password, url, connectionName string) (*bytes.Buffer, error) {
	body := connectionCreatePost{
		Name: connectionName,
		ConfiguredProperties: configuredProperties{
			SkipCertificateCheck: "true",
			ConnectionURI:        url,
			Username:             username,
			Password:             password,
		},
		Connector: connector{
			Description:      "Subscribe for and publish messages.",
			Icon:             "fa-amqp",
			ComponentScheme:  "amqp",
			ConnectorFactory: "io.syndesis.connector.amqp.AMQPConnectorFactory",
			Tags: []string{
				"verifier",
			},
			Uses: 0,
			ActionsSummary: connectorActionsSummary{
				TotalActions: 3,
			},
			ID:      "amqp",
			Version: 3,
			Actions: []connectorAction{
				{
					ID:          "io.syndesis:amqp-publish-action",
					Name:        "Publish messages",
					Description: "Send data to the destination you specify.",
					Descriptor: connectorActionDescriptor{
						InputDataShape: connectorDataShape{
							Kind: "any",
						},
						OutputDataShape: connectorDataShape{
							Kind: "none",
						},
						PropertyDefinitionSteps: []connectorStep{
							{
								Description: "Specify AMQP destination properties, including Queue or Topic name",
								Name:        "Select the AMQP Destination",
								Properties: map[string]connectorProperty{
									"destinationName": connectorProperty{
										ComponentProperty: false,
										Deprecated:        false,
										LabelHint:         "Name of the queue or topic to send data to.",
										DisplayName:       "Destination Name",
										Group:             "common",
										JavaType:          "java.lang.String",
										Kind:              "path",
										Required:          true,
										Secret:            false,
										Type:              "string",
										Order:             1,
									},
									"destinationType": connectorProperty{
										ComponentProperty: false,
										DefaultValue:      "queue",
										Deprecated:        false,
										LabelHint:         "By default, the destination is a Queue.",
										DisplayName:       "Destination Type",
										Group:             "common",
										JavaType:          "java.lang.String",
										Kind:              "path",
										Required:          false,
										Secret:            false,
										Type:              "string",
										Order:             2,
										Enum: []connectorEnum{
											{
												Label: "Topic",
												Value: "topic",
											},
											{
												Label: "Queue",
												Value: "queue",
											},
										},
									},
									"deliveryPersistent": connectorProperty{
										ComponentProperty: false,
										DefaultValue:      "true",
										Deprecated:        false,
										LabelHint:         "Message delivery is guaranteed when Persistent is selected.",
										DisplayName:       "Persistent",
										Group:             "producer",
										JavaType:          "boolean",
										Kind:              "parameter",
										Label:             "producer",
										Required:          false,
										Secret:            false,
										Type:              "boolean",
										Order:             3,
									},
								},
							},
						},
					},
					ActionType: "connector",
					Pattern:    "To",
				}, {
					ID:          "io.syndesis:amqp-subscribe-action",
					Name:        "Subscribe for messages",
					Description: "Receive data from the destination you specify.",
					Descriptor: connectorActionDescriptor{
						InputDataShape: connectorDataShape{
							Kind: "none",
						},
						OutputDataShape: connectorDataShape{
							Kind: "any",
						},
						PropertyDefinitionSteps: []connectorStep{
							{
								Description: "Specify AMQP destination properties, including Queue or Topic Name",
								Name:        "Select the AMQP Destination",
								Properties: map[string]connectorProperty{
									"destinationName": connectorProperty{
										ComponentProperty: false,
										Deprecated:        false,
										LabelHint:         "Name of the queue or topic to receive data from.",
										DisplayName:       "Destination Name",
										Group:             "common",
										JavaType:          "java.lang.String",
										Kind:              "path",
										Required:          true,
										Secret:            false,
										Type:              "string",
										Order:             1,
									},
									"destinationType": connectorProperty{
										ComponentProperty: false,
										DefaultValue:      "queue",
										Deprecated:        false,
										LabelHint:         "By default, the destination is a Queue.",
										DisplayName:       "Destination Type",
										Group:             "common",
										JavaType:          "java.lang.String",
										Kind:              "path",
										Required:          false,
										Secret:            false,
										Type:              "string",
										Order:             2,
										Enum: []connectorEnum{
											{
												Label: "Topic",
												Value: "topic",
											},
											{
												Label: "Queue",
												Value: "queue",
											},
										},
									},
									"durableSubscriptionId": connectorProperty{
										ComponentProperty: false,
										Deprecated:        false,
										LabelHint:         "Set the ID that lets connections close and reopen with missing messages. Connection type must be a topic.",
										DisplayName:       "Durable Subscription ID",
										Group:             "consumer",
										JavaType:          "java.lang.String",
										Kind:              "parameter",
										Label:             "consumer",
										Required:          false,
										Secret:            false,
										Type:              "string",
										Order:             3,
									},
									"messageSelector": connectorProperty{
										ComponentProperty: false,
										Deprecated:        false,
										LabelHint:         "Specify a filter expression to receive only data that meets certain criteria.",
										DisplayName:       "Message Selector",
										Group:             "consumer (advanced)",
										JavaType:          "java.lang.String",
										Kind:              "parameter",
										Label:             "consumer,advanced",
										Required:          false,
										Secret:            false,
										Type:              "string",
										Order:             4,
									},
								},
							},
						},
					},
					ActionType: "connector",
					Pattern:    "From",
				}, {
					ID:          "io.syndesis:amqp-request-action",
					Name:        "Request response using messages",
					Description: "Send data to the destination you specify and receive a response.",
					Descriptor: connectorActionDescriptor{
						InputDataShape: connectorDataShape{
							Kind: "any",
						},
						OutputDataShape: connectorDataShape{
							Kind: "any",
						},
						PropertyDefinitionSteps: []connectorStep{
							{
								Description: "Specify AMQP destination properties, including Queue or Topic Name",
								Name:        "Select the AMQP Destination",
								Properties: map[string]connectorProperty{
									"destinationName": connectorProperty{
										ComponentProperty: false,
										Deprecated:        false,
										LabelHint:         "Name of the queue or topic to receive data from.",
										DisplayName:       "Destination Name",
										Group:             "common",
										JavaType:          "java.lang.String",
										Kind:              "path",
										Required:          true,
										Secret:            false,
										Type:              "string",
										Order:             1,
									},
									"destinationType": connectorProperty{
										ComponentProperty: false,
										DefaultValue:      "queue",
										Deprecated:        false,
										LabelHint:         "By default, the destination is a Queue.",
										DisplayName:       "Destination Type",
										Group:             "common",
										JavaType:          "java.lang.String",
										Kind:              "path",
										Required:          false,
										Secret:            false,
										Type:              "string",
										Order:             2,
										Enum: []connectorEnum{
											{
												Label: "Topic",
												Value: "topic",
											},
											{
												Label: "Queue",
												Value: "queue",
											},
										},
									},
									"durableSubscriptionId": connectorProperty{
										ComponentProperty: false,
										Deprecated:        false,
										LabelHint:         "Set the ID that lets connections close and reopen with missing messages. Connection type must be a topic.",
										DisplayName:       "Durable Subscription ID",
										Group:             "consumer",
										JavaType:          "java.lang.String",
										Kind:              "parameter",
										Label:             "consumer",
										Required:          false,
										Secret:            false,
										Type:              "string",
										Order:             3,
									},
									"messageSelector": connectorProperty{
										ComponentProperty: false,
										Deprecated:        false,
										LabelHint:         "Specify a filter expression to receive only data that meets certain criteria.",
										DisplayName:       "Message Selector",
										Group:             "consumer (advanced)",
										JavaType:          "java.lang.String",
										Kind:              "parameter",
										Label:             "consumer,advanced",
										Required:          false,
										Secret:            false,
										Type:              "string",
										Order:             4,
									},
								},
							},
						},
					},
					ActionType: "connector",
					Pattern:    "To",
				},
			},
			Name: "AMQP Message Broker",
			Properties: map[string]connectorProperty{
				"connectionUri": connectorProperty{
					ComponentProperty: true,
					Deprecated:        false,
					LabelHint:         "Location to send data to or obtain data from.",
					DisplayName:       "Connection URI",
					Group:             "common",
					JavaType:          "java.lang.String",
					Kind:              "property",
					Label:             "common",
					Required:          true,
					Secret:            false,
					Type:              "string",
					Order:             1,
				},
				"username": connectorProperty{
					ComponentProperty: true,
					Deprecated:        false,
					LabelHint:         "Access the broker with this user’s authorization credentials.",
					DisplayName:       "User Name",
					Group:             "security",
					JavaType:          "java.lang.String",
					Kind:              "property",
					Label:             "common,security",
					Required:          false,
					Secret:            false,
					Type:              "string",
					Order:             2,
				},
				"password": connectorProperty{
					ComponentProperty: true,
					Deprecated:        false,
					LabelHint:         "Password for the specified user account.",
					DisplayName:       "Password",
					Group:             "security",
					JavaType:          "java.lang.String",
					Kind:              "property",
					Label:             "common,security",
					Required:          false,
					Secret:            true,
					Type:              "string",
					Order:             3,
				},
				"clientID": connectorProperty{
					ComponentProperty: true,
					Deprecated:        false,
					LabelHint:         "Required for connections to close and reopen without missing messages. Connection destination must be a topic.",
					DisplayName:       "Client ID",
					Group:             "security",
					JavaType:          "java.lang.String",
					Kind:              "property",
					Label:             "common,security",
					Required:          false,
					Secret:            false,
					Type:              "string",
					Order:             4,
				},
				"skipCertificateCheck": connectorProperty{
					ComponentProperty: true,
					DefaultValue:      "false",
					Deprecated:        false,
					LabelHint:         "Ensure certificate checks are enabled for secure production environments. Disable for convenience in only development environments.",
					DisplayName:       "Check Certificates",
					Group:             "security",
					JavaType:          "java.lang.String",
					Kind:              "property",
					Label:             "common,security",
					Required:          false,
					Secret:            false,
					Type:              "string",
					Order:             5,
					Enum: []connectorEnum{
						{
							Label: "Disable",
							Value: "true",
						},
						{
							Label: "Enable",
							Value: "false",
						},
					},
				},
				"brokerCertificate": connectorProperty{
					ComponentProperty: true,
					Deprecated:        false,
					Description:       "AMQ Broker X.509 PEM Certificate",
					DisplayName:       "Broker Certificate",
					Group:             "security",
					JavaType:          "java.lang.String",
					Kind:              "property",
					Label:             "common,security",
					Required:          false,
					Secret:            false,
					Type:              "textarea",
					Order:             6,
					Relation: []connectorPropertyRelation{
						{
							Action: "ENABLE",
							When: []connectorPropertyEvent{
								{
									ID:    "skipCertificateCheck",
									Value: "false",
								},
							},
						},
					},
				},
				"clientCertificate": connectorProperty{
					ComponentProperty: true,
					Deprecated:        false,
					Description:       "AMQ Client X.509 PEM Certificate",
					DisplayName:       "Client Certificate",
					Group:             "security",
					JavaType:          "java.lang.String",
					Kind:              "property",
					Label:             "common,security",
					Required:          false,
					Secret:            false,
					Type:              "textarea",
					Order:             7,
					Relation: []connectorPropertyRelation{
						{
							Action: "ENABLE",
							When: []connectorPropertyEvent{
								{
									ID:    "skipCertificateCheck",
									Value: "false",
								},
							},
						},
					},
				},
			},
			Dependencies: []connectorDependency{
				{
					Type: "MAVEN",
					ID:   "io.syndesis.connector:connector-amqp:1.5-SNAPSHOT",
				},
			},
		},
		Icon:        "fa-amqp",
		ConnectorID: "amqp",
		Description: "",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(bodyBytes), nil
}

type connectionCreatePost struct {
	ConfiguredProperties configuredProperties `json:"configuredProperties"`
	Name                 string               `json:"name"`
	Description          string               `json:"description"`
	Connector            connector            `json:"connector"`
	Icon                 string               `json:"icon"`
	ConnectorID          string               `json:"connectorId"`
}

type connector struct {
	Tags             []string                     `json:"tags,omitempty"`
	Uses             int                          `json:"uses"`
	Description      string                       `json:"description"`
	Name             string                       `json:"name,omitempty"`
	Icon             string                       `json:"icon"`
	ComponentScheme  string                       `json:"componentScheme"`
	ConnectorFactory string                       `json:"connectorFactory,omitempty"`
	ID               string                       `json:"id"`
	Version          int                          `json:"version"`
	Dependencies     []connectorDependency        `json:"dependencies"`
	Actions          []connectorAction            `json:"actions"`
	Properties       map[string]connectorProperty `json:"properties"`
	ActionsSummary   connectorActionsSummary      `json:"actionsSummary"`
}

type connectorActionsSummary struct {
	TotalActions int `json:"totalActions"`
}
type connectorAction struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Descriptor  connectorActionDescriptor `json:"descriptor"`
	ActionType  string                    `json:"actionType"`
	Pattern     string                    `json:"pattern"`
}

type connectorDataShape struct {
	Kind string `json:"kind"`
}

type connectorEnum struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type connectorStep struct {
	Description string                       `json:"description"`
	Name        string                       `json:"name"`
	Properties  map[string]connectorProperty `json:"properties"`
}

type connectorProperty struct {
	ComponentProperty bool                        `json:"componentProperty"`
	Deprecated        bool                        `json:"deprecated"`
	Description       string                      `json:"description,omitempty"`
	LabelHint         string                      `json:"labelHint,omitempty"`
	DisplayName       string                      `json:"displayName,omitempty"`
	Group             string                      `json:"group,omitempty"`
	JavaType          string                      `json:"javaType,omitempty"`
	Kind              string                      `json:"kind,omitempty"`
	Label             string                      `json:"label,omitempty"`
	Required          bool                        `json:"required"`
	Secret            bool                        `json:"secret"`
	Type              string                      `json:"type,omitempty"`
	Order             int                         `json:"order,omitempty"`
	Relation          []connectorPropertyRelation `json:"relation,omitempty"`
	Enum              []connectorEnum             `json:"enum,omitempty"`
	DefaultValue      string                      `json:"defaultValue,omitempty"`
	PlaceHolder       string                      `json:"placeholder,omitempty"`
}

type connectorPropertyEvent struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type connectorPropertyRelation struct {
	Action string                   `json:"action"`
	When   []connectorPropertyEvent `json:"when"`
}

type connectorActionDescriptor struct {
	InputDataShape          connectorDataShape `json:"inputDataShape"`
	OutputDataShape         connectorDataShape `json:"outputDataShape"`
	PropertyDefinitionSteps []connectorStep    `json:"propertyDefinitionSteps"`
	ConnectorFactory        string             `json:"connectorFactory,omitempty"`
}

type connectorDependency struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
type configuredProperties struct {
	ConnectionURI        string `json:"connectionUri,omitempty"`
	Username             string `json:"username,omitempty"`
	Password             string `json:"password,omitempty"`
	SkipCertificateCheck string `json:"skipCertificateCheck,omitempty"`
	BrokerCertificate    string `json:"brokerCertificate,omitempty"`
	BaseURL              string `json:"baseUrl,omitempty"`
}

type syndesisClient struct {
	token     string
	host      string
	user      string
	xsrfToken string
	client    *http.Client
}

type connectionCreationResponse struct {
	ID string `json:"id"`
}
