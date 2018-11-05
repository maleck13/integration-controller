package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/typed/core/v1"

	syndesis "github.com/integr8ly/integration-controller/pkg/apis/syndesis/v1alpha1"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate moq -out requester_mock_test.go . httpRequester
type httpRequester interface {
	Do(r *http.Request) (*http.Response, error)
}

type credentials struct {
	user string
	pass string
}

type Cruder struct {
	namespace    string
	httpClient   httpRequester
	saUser       string
	saToken      string
	xsrfToken    string
	servicePort  string
	secretClient v1.SecretInterface
}

func responseCloser(closer io.Closer) {
	if err := closer.Close(); err != nil {
		logrus.Error("failed to close response body")
	}
}

func defaultHTTPRequester() httpRequester {
	client := http.DefaultClient
	client.Timeout = 5 * time.Second
	return client
}

func New(httpClient httpRequester, secretClient v1.SecretInterface, namespace, saToken, saUser string) *Cruder {
	cc := &Cruder{
		namespace:    namespace,
		saToken:      saToken,
		saUser:       saUser,
		httpClient:   httpClient,
		secretClient: secretClient,
		xsrfToken:    "awesome",
	}
	if httpClient == nil {
		cc.httpClient = defaultHTTPRequester()
	}
	return cc
}

func (cc *Cruder) UpdateConnection(c *syndesis.Connection) (*syndesis.Connection, error) {
	return c, nil
}

func (cc *Cruder) host() string {
	if os.Getenv("LOCAL_DEV") != "" {
		return "localhost:8145"
	}
	return "syndesis-server." + cc.namespace + ".svc"
}

func (cc *Cruder) getCredentials(secretName string) (*credentials, error) {
	s, err := cc.secretClient.Get(secretName, v12.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get credentials secret")
	}
	var cred credentials
	if _, ok := s.StringData["user"]; !ok {
		return nil, errors.New("expected to find a user key in the secret")
	}
	if _, ok := s.StringData["pass"]; !ok {
		return nil, errors.New("expected to find a pass key in the secret")
	}
	cred.user = s.StringData["user"]
	cred.pass = s.StringData["pass"]
	return &cred, nil
}

func (cc *Cruder) ConnectionExists(c *syndesis.Connection) (bool, error) {
	host := cc.host()
	if c.Status.SyndesisID == "" {
		return false, nil
	}
	resource := "connections"
	if c.Spec.Type == "api" {
		resource = "connectors"
	}
	get := fmt.Sprintf("http://%s/api/v1/%s/%s", host, resource, c.Status.SyndesisID)
	logrus.Debug("checking connection exits at ", get)
	req, err := http.NewRequest("GET", get, nil)
	if err != nil {
		return false, err
	}
	cc.setHeaders(req)
	resp, err := cc.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer responseCloser(resp.Body)
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, errors.New("unexpected response code " + resp.Status)
}

//TODO these methods probably should take a customisation object

func (cc *Cruder) CreateCustomisation(c *syndesis.Connection) (*syndesis.Connection, error) {
	logrus.Debug("CreateCustomisation")
	info := apiConnectorInfo{
		ConnectorTemplateID:  "swagger-connector-template",
		ConfiguredProperties: configProperties{Specification: c.Spec.URL},
	}
	body := bytes.Buffer{}
	encoder := json.NewEncoder(&body)
	if err := encoder.Encode(&info); err != nil {
		return c, err
	}
	connector := &apiConnector{}
	if err := cc.post("/api/v1/connectors/custom/info", &body, func(resp *http.Response) error {
		logrus.Debug("** INFO CALLBACK CALLED")
		if resp.StatusCode != http.StatusOK {
			return errors.New("unexpected response code from fuse online /api/v1/connectors/custom/info " + resp.Status)
		}
		//data, _ := ioutil.ReadAll(resp.Body)
		//logrus.Debug("info response ", string(data))
		if err := json.NewDecoder(resp.Body).Decode(connector); err != nil {
			return err
		}
		// check for errors in the response
		return nil
	}); err != nil {
		return c, err
	}
	body.Reset()
	connector.ConnectorTemplateID = info.ConnectorTemplateID
	if err := encoder.Encode(connector); err != nil {
		return c, err
	}
	logrus.Debug("connector body response ", string(body.Bytes()))
	var connResp = &connectionCreationResponse{}
	if err := cc.post("/api/v1/connectors/custom", &body, func(resp *http.Response) error {
		logrus.Debug("custom connector callback ", resp.Status)
		if resp.StatusCode != http.StatusOK {
			return errors.New("unexpected response code from fuse online /api/v1/connectors/custom " + resp.Status)
		}

		if err := json.NewDecoder(resp.Body).Decode(connResp); err != nil {
			return err
		}
		logrus.Debug("custom connector response ", connResp)
		return nil
	}); err != nil {
		return c, err
	}
	c.Status.SyndesisID = connResp.ID
	return c, nil
}

func (cc *Cruder) post(path string, body io.Reader, responseHandler func(resp *http.Response) error) error {
	host := cc.host()
	apiHost := "http://" + host + path
	var (
		err error
	)
	req, err := http.NewRequest("POST", apiHost, body)
	if err != nil {
		return err
	}
	cc.setHeaders(req)
	rsp, err := cc.httpClient.Do(req)
	defer responseCloser(rsp.Body)
	if err == nil {
		return responseHandler(rsp)
	}
	return err
}

func (cc *Cruder) CreateConnection(c *syndesis.Connection) (*syndesis.Connection, error) {
	host := cc.host()
	apiHost := "http://" + host + "/api/v1/connections"
	var (
		body *bytes.Buffer
		err  error
	)
	var (
		creds   *credentials
		credErr error
	)
	if c.Spec.Credentials != "" {
		creds, credErr = cc.getCredentials(c.Spec.Credentials)
		if credErr != nil {
			return nil, credErr
		}
	}

	switch c.Spec.Type {
	case "amqp":
		logrus.Debug("creating new AMQP connection: %s", c.Name)
		body, err = newAMQPConnectionCreatePostBody(creds.user, creds.pass, c.Spec.URL, c.Name)
		if err != nil {
			return c, err
		}
	case "http":
		logrus.Debug("creating new HTTP connection: %s", c.Name)
		body, err = newHTTPConnectionCreatePostBody(c.Spec.URL, c.Name, "auto discovered http connection")
		if err != nil {
			return c, err
		}
	default:
		return c, fmt.Errorf("unknown connection type '%s'", c.Spec.Type)
	}

	req, err := http.NewRequest("POST", apiHost, body)
	if err != nil {
		return c, err
	}
	cc.setHeaders(req)
	rsp, err := cc.httpClient.Do(req)
	if err != nil {
		if rsp != nil && rsp.Body != nil {
			rspBodyBytes, _ := ioutil.ReadAll(rsp.Body)
			logrus.Infof("response to bad create request:: %s", string(rspBodyBytes))
		}
		return c, err
	}

	defer responseCloser(rsp.Body)
	if rsp.StatusCode != 200 {
		logrus.Infof("request headers: %+v", req.Header)
		logrus.Info("response status from fuse ", rsp.Status)
		rspBodyBytes, _ := ioutil.ReadAll(rsp.Body)
		logrus.Infof("response to bad create request:: %s", string(rspBodyBytes))
		return c, fmt.Errorf("error creating connection")
	}

	dec := json.NewDecoder(rsp.Body)
	connResp := &connectionCreationResponse{}
	if err := dec.Decode(connResp); err != nil {
		return c, errors.Wrap(err, " failed to decode connection creation response")
	}
	c.Status.SyndesisID = connResp.ID
	return c, nil

}

func (cc *Cruder) DeleteConnector(c *syndesis.Connection) error {
	host := cc.host()
	logrus.Info("Deleting connection")
	apiHost := "http://" + host + "/api/v1/connectors/" + c.Status.SyndesisID
	req, err := http.NewRequest("DELETE", apiHost, nil)
	if err != nil {
		return err
	}
	cc.setHeaders(req)
	resp, err := cc.httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return errors.New("unexpected response from fuse api status code " + resp.Status)
	}
	return nil
}

func (cc *Cruder) DeleteConnection(c *syndesis.Connection) error {
	host := cc.host()
	logrus.Info("Deleting connection")
	apiHost := "http://" + host + "/api/v1/connections/" + c.Status.SyndesisID
	req, err := http.NewRequest("DELETE", apiHost, nil)
	if err != nil {
		return err
	}
	cc.setHeaders(req)
	resp, err := cc.httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return errors.New("unexpected response from fuse api status code " + resp.Status)
	}
	return nil
}

func (cc *Cruder) setHeaders(req *http.Request) {
	req.Header.Set("X-FORWARDED-USER", cc.saUser)
	req.Header.Set("X-FORWARDED-ACCESS-TOKEN", cc.saToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("SYNDESIS-XSRF-TOKEN", cc.xsrfToken)
}

func newHTTPConnectionCreatePostBody(url, name, description string) (*bytes.Buffer, error) {
	body := connectionCreatePost{
		Connector:   nil,
		Icon:        "http",
		ConnectorID: "http4",
		ConfiguredProperties: configuredProperties{
			BaseURL: url,
		},
		Name:        name,
		Description: description,
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
		Connector:   nil,
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
	Connector            *connector           `json:"-"`
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

type apiConnector struct {
	ConnectorTemplateID string `json:"connectorTemplateId"`
	IsComplete          bool   `json:"isComplete"`
	IsOK                bool   `json:"isOK"`
	IsRequested         bool   `json:"isRequested"`
	Errors              []struct {
		Error    string `json:"error"`
		Message  string `json:"message"`
		Property string `json:"property"`
	} `json:"errors"`
	SpecificationFile    interface{} `json:"specificationFile"`
	ConfiguredProperties struct {
		Specification      string `json:"specification"`
		AuthenticationType string `json:"authenticationType"`
		Host               string `json:"host"`
		BasePath           string `json:"basePath"`
	} `json:"configuredProperties"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	ActionsSummary struct {
		ActionCountByTags struct {
			Flights int `json:"flights"`
		} `json:"actionCountByTags"`
		TotalActions int `json:"totalActions"`
	} `json:"actionsSummary"`
	Properties struct {
		AuthenticationType struct {
			ComponentProperty bool     `json:"componentProperty"`
			DefaultValue      string   `json:"defaultValue"`
			Deprecated        bool     `json:"deprecated"`
			Description       string   `json:"description"`
			DisplayName       string   `json:"displayName"`
			Group             string   `json:"group"`
			JavaType          string   `json:"javaType"`
			Kind              string   `json:"kind"`
			Label             string   `json:"label"`
			Required          bool     `json:"required"`
			Secret            bool     `json:"secret"`
			Type              string   `json:"type"`
			Tags              []string `json:"tags"`
			Order             int      `json:"order"`
			Enum              []struct {
				Label string `json:"label"`
				Value string `json:"value"`
			} `json:"enum"`
		} `json:"authenticationType"`
		BasePath struct {
			ComponentProperty bool   `json:"componentProperty"`
			DefaultValue      string `json:"defaultValue"`
			Deprecated        bool   `json:"deprecated"`
			Description       string `json:"description"`
			DisplayName       string `json:"displayName"`
			Group             string `json:"group"`
			JavaType          string `json:"javaType"`
			Kind              string `json:"kind"`
			Label             string `json:"label"`
			Required          bool   `json:"required"`
			Secret            bool   `json:"secret"`
			Type              string `json:"type"`
			Order             int    `json:"order"`
		} `json:"basePath"`
		Host struct {
			ComponentProperty bool   `json:"componentProperty"`
			DefaultValue      string `json:"defaultValue"`
			Deprecated        bool   `json:"deprecated"`
			Description       string `json:"description"`
			DisplayName       string `json:"displayName"`
			Group             string `json:"group"`
			JavaType          string `json:"javaType"`
			Kind              string `json:"kind"`
			Label             string `json:"label"`
			Required          bool   `json:"required"`
			Secret            bool   `json:"secret"`
			Type              string `json:"type"`
			Order             int    `json:"order"`
		} `json:"host"`
		Specification struct {
			ComponentProperty bool     `json:"componentProperty"`
			Deprecated        bool     `json:"deprecated"`
			Description       string   `json:"description"`
			DisplayName       string   `json:"displayName"`
			Group             string   `json:"group"`
			JavaType          string   `json:"javaType"`
			Kind              string   `json:"kind"`
			Label             string   `json:"label"`
			Required          bool     `json:"required"`
			Secret            bool     `json:"secret"`
			Type              string   `json:"type"`
			Tags              []string `json:"tags"`
		} `json:"specification"`
	} `json:"properties"`
}

type apiConnectorResp struct {
	ActionsSummary struct {
		ActionCountByTags map[string]int `json:"actionCountByTags"`
		TotalActions      int            `json:"totalActions"`
	} `json:"actionsSummary"`
	Description string `json:"description"`
	Errors      []struct {
		Error    string `json:"error"`
		Message  string `json:"message"`
		Property string `json:"property"`
	} `json:"errors"`
	Icon       string `json:"icon"`
	Name       string `json:"name"`
	Properties struct {
		AuthenticationType struct {
			ComponentProperty bool     `json:"componentProperty"`
			DefaultValue      string   `json:"defaultValue"`
			Deprecated        bool     `json:"deprecated"`
			Description       string   `json:"description"`
			DisplayName       string   `json:"displayName"`
			Group             string   `json:"group"`
			JavaType          string   `json:"javaType"`
			Kind              string   `json:"kind"`
			Label             string   `json:"label"`
			Required          bool     `json:"required"`
			Secret            bool     `json:"secret"`
			Type              string   `json:"type"`
			Tags              []string `json:"tags"`
			Order             int      `json:"order"`
			Enum              []struct {
				Label string `json:"label"`
				Value string `json:"value"`
			} `json:"enum"`
		} `json:"authenticationType"`
		BasePath struct {
			ComponentProperty bool   `json:"componentProperty"`
			DefaultValue      string `json:"defaultValue"`
			Deprecated        bool   `json:"deprecated"`
			Description       string `json:"description"`
			DisplayName       string `json:"displayName"`
			Group             string `json:"group"`
			JavaType          string `json:"javaType"`
			Kind              string `json:"kind"`
			Label             string `json:"label"`
			Required          bool   `json:"required"`
			Secret            bool   `json:"secret"`
			Type              string `json:"type"`
			Order             int    `json:"order"`
		} `json:"basePath"`
		Host struct {
			ComponentProperty bool   `json:"componentProperty"`
			DefaultValue      string `json:"defaultValue"`
			Deprecated        bool   `json:"deprecated"`
			Description       string `json:"description"`
			DisplayName       string `json:"displayName"`
			Group             string `json:"group"`
			JavaType          string `json:"javaType"`
			Kind              string `json:"kind"`
			Label             string `json:"label"`
			Required          bool   `json:"required"`
			Secret            bool   `json:"secret"`
			Type              string `json:"type"`
			Order             int    `json:"order"`
		} `json:"host"`
		Specification struct {
			ComponentProperty bool     `json:"componentProperty"`
			Deprecated        bool     `json:"deprecated"`
			Description       string   `json:"description"`
			DisplayName       string   `json:"displayName"`
			Group             string   `json:"group"`
			JavaType          string   `json:"javaType"`
			Kind              string   `json:"kind"`
			Label             string   `json:"label"`
			Required          bool     `json:"required"`
			Secret            bool     `json:"secret"`
			Type              string   `json:"type"`
			Tags              []string `json:"tags"`
		} `json:"specification"`
	} `json:"properties"`
	ConfiguredProperties struct {
		Specification string `json:"specification"`
	} `json:"configuredProperties"`
}

type apiConnectorInfo struct {
	ConnectorTemplateID  string           `json:"connectorTemplateId"`
	IsComplete           bool             `json:"isComplete"`
	IsOK                 bool             `json:"isOK"`
	IsRequested          bool             `json:"isRequested"`
	Errors               []interface{}    `json:"errors"`
	Warnings             []interface{}    `json:"warnings"`
	SpecificationFile    interface{}      `json:"specificationFile"`
	ConfiguredProperties configProperties `json:"configuredProperties"`
	Name                 interface{}      `json:"name"`
	Description          interface{}      `json:"description"`
}

type configProperties struct {
	Specification string `json:"specification"`
}
