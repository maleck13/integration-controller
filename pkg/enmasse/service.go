package enmasse

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	errors2 "github.com/integr8ly/integration-controller/pkg/errors"

	"github.com/integr8ly/integration-controller/pkg/apis/integration/v1alpha1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	authUrl = "auth/realms/master/protocol/openid-connect/token"
)

type Service struct {
	k8sclient   kubernetes.Interface
	routeClient routev1.RouteInterface
	ns          string
}

func NewService(k8sclient kubernetes.Interface, routeClient routev1.RouteInterface, targetNS string) *Service {
	return &Service{k8sclient: k8sclient, routeClient: routeClient, ns: targetNS}
}

func (s *Service) CreateUser(userName, realm string) (*v1alpha1.User, error) {
	// note in the next release of enmasse we will be able to replace this with a crd creation
	route, err := s.routeClient.Get("keycloak", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list routes for enmasse")
	}
	logrus.Info("found route for keycloak ", route.Name)
	// find secret for keycloak
	cred, err := s.k8sclient.CoreV1().Secrets(s.ns).Get("keycloak-credentials", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get keycloak credentials for enmasse")
	}
	pass := string(cred.Data["admin.password"])
	user := string(cred.Data["admin.username"])
	host := "https://" + route.Spec.Host
	authToken, err := keycloakLogin("https://"+route.Spec.Host, string(user), string(pass))
	if err != nil {
		return nil, errors.Wrap(err, "failed to login to enmasse keycloak")
	}
	u, err := createUser(host, authToken, realm, userName, pass)
	if err != nil && errors2.IsAlreadyExistsErr(err) {
		return &v1alpha1.User{Password: pass, UserName: userName}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new user for enmasse")
	}
	return u, nil
}

func defaultRequester() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c := &http.Client{Transport: transport, Timeout: time.Second * 10}
	return c
}

func keycloakLogin(host, user, pass string) (string, error) {
	form := url.Values{}
	form.Add("username", user)
	form.Add("password", pass)
	form.Add("client_id", "admin-cli")
	form.Add("grant_type", "password")

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/%s", host, authUrl),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", errors.Wrap(err, "error creating login request")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res, err := defaultRequester().Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return "", errors.Wrap(err, "error performing token request")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logrus.Errorf("error reading response %+v", err)
		return "", errors.Wrap(err, "error reading token response")
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.New("unexpected status code when logging in" + res.Status)
	}
	tokenRes := &tokenResponse{}
	err = json.Unmarshal(body, tokenRes)
	if err != nil {
		logrus.Error("failed to parse body ", string(body), res.StatusCode)
		return "", errors.Wrap(err, "error parsing token response ")
	}

	if tokenRes.Error != "" {
		logrus.Errorf("error with request: " + tokenRes.ErrorDescription)
		return "", errors.New(tokenRes.ErrorDescription)
	}

	return tokenRes.AccessToken, nil

}

func createUser(host, token, realm, userName, password string) (*v1alpha1.User, error) {
	//https://keycloak-john-integration.apps.sedroche.openshiftworkshop.com/auth/admin/realms/john-integration-messaging-service/users
	//https://keycloak-john-integration.apps.sedroche.openshiftworkshop.com/auth/admin/realms/John-integration-messaging-service/users
	u := "%s/auth/admin/realms/%s/users"
	url := fmt.Sprintf(u, host, realm)
	logrus.Debug("calling keycloak url: ", url)
	user := &keycloakApiUser{
		UserName: userName,
		Credentials: []credential{
			{
				Type:      "password",
				Value:     password,
				Temporary: false,
			},
		},
		Enabled: true,
		RealmRoles: []string{
			"offline_access",
			"uma_authorization",
		},
		Groups: []string{
			"send_*",
			"recv_*",
			"monitor",
			"manage",
		},
	}
	body, err := json.Marshal(user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request body for creating new enmasse user")
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request to create new user")
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	res, err := defaultRequester().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request to create new enmasse user")
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusConflict {
		logrus.Info("user already exists doing nothing")
		return nil, &errors2.AlreadyExistsErr{}

	}
	if res.StatusCode != http.StatusCreated {
		return nil, errors.New("failed to create user status code " + res.Status)
	}
	return &v1alpha1.User{UserName: userName, Password: password}, nil
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type keycloakApiUser struct {
	ID              string              `json:"id,omitempty"`
	UserName        string              `json:"username,omitempty"`
	FirstName       string              `json:"firstName"`
	LastName        string              `json:"lastName"`
	Email           string              `json:"email,omitempty"`
	EmailVerified   bool                `json:"emailVerified"`
	Enabled         bool                `json:"enabled"`
	RealmRoles      []string            `json:"realmRoles,omitempty"`
	ClientRoles     map[string][]string `json:"clientRoles"`
	RequiredActions []string            `json:"requiredActions,omitempty"`
	Groups          []string            `json:"groups,omitempty"`
	Credentials     []credential        `json:"credentials"`
}

type credential struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}
