package enmasse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/integr8ly/integration-controller/pkg/transport"

	errors3 "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/api/core/v1"

	"github.com/pborman/uuid"

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
	routeClient routev1.RoutesGetter
	enmasseNS   string
	currentNS   string
	httpClient  *http.Client
}

func NewService(k8sclient kubernetes.Interface, routeClient routev1.RoutesGetter, httpClient *http.Client, targetNS string) *Service {
	return &Service{k8sclient: k8sclient, routeClient: routeClient, enmasseNS: targetNS, httpClient: httpClient}
}

func (s *Service) DeleteUser(userName, realm string) error {
	route, err := s.routeClient.Routes(s.enmasseNS).Get("keycloak", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list routes for enmasse")
	}
	logrus.Info("found route for keycloak ", route.Name)
	// find secret for keycloak
	cred, err := s.k8sclient.CoreV1().Secrets(s.enmasseNS).Get("keycloak-credentials", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get keycloak credentials for enmasse")
	}
	pass := string(cred.Data["admin.password"])
	user := string(cred.Data["admin.username"])
	host := "https://" + route.Spec.Host
	authToken, err := s.keycloakLogin("https://"+route.Spec.Host, string(user), string(pass))
	if err != nil {
		return errors.Wrap(err, "failed to login to enmasse keycloak")
	}
	id, err := s.getUserID(host, authToken, realm, userName)
	if err != nil {
		return err
	}
	u := fmt.Sprintf("%s/auth/admin/realms/%s/users/%s", host, realm, id)
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to delete user from keycloak")
	}
	defer transport.ResponseCloser(resp)
	if resp.StatusCode != http.StatusNoContent {
		return errors.New("unexpected response code from keycloak " + resp.Status)
	}
	if err := s.k8sclient.CoreV1().Secrets(s.currentNS).Delete(realm+"-"+userName, metav1.NewDeleteOptions(0)); err != nil && !errors3.IsNotFound(err) {
		logrus.Error("failed to clean up user secret", err)
	}
	return nil
}

func (s *Service) CreateUser(userName, realm string) (*v1alpha1.User, error) {
	// note in the next release of enmasse we will be able to replace this with a crd creation
	route, err := s.routeClient.Routes(s.enmasseNS).Get("keycloak", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list routes for enmasse")
	}
	logrus.Info("found route for keycloak ", route.Name)
	// find secret for keycloak
	cred, err := s.k8sclient.CoreV1().Secrets(s.enmasseNS).Get("keycloak-credentials", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get keycloak credentials for enmasse")
	}
	pass := string(cred.Data["admin.password"])
	user := string(cred.Data["admin.username"])
	host := "https://" + route.Spec.Host

	authToken, err := s.keycloakLogin("https://"+route.Spec.Host, string(user), string(pass))
	if err != nil {
		return nil, errors.Wrap(err, "failed to login to enmasse keycloak")
	}
	userPass := uuid.New()
	u, err := s.createUser(host, authToken, realm, userName, userPass)
	secretName := realm + "-" + userName
	if err != nil && errors2.IsAlreadyExistsErr(err) {
		logrus.Debug("enmasse keycloak user already exists reading credentials from secret")
		us, err := s.k8sclient.CoreV1().Secrets(s.currentNS).Get(secretName, metav1.GetOptions{})
		if err != nil {

		}
		return &v1alpha1.User{Password: us.StringData["pass"], UserName: us.StringData["user"]}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new user for enmasse")
	}
	// add a secret with this users details
	us := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName},
		StringData: map[string]string{"user": u.UserName, "pass": u.Password}}
	if _, err := s.k8sclient.CoreV1().Secrets(s.currentNS).Create(us); err != nil {
		logrus.Error("failed to store user credentials in a secret ", err)
	}
	return u, nil
}

func (s *Service) keycloakLogin(host, user, pass string) (string, error) {
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
	res, err := s.httpClient.Do(req)
	if err != nil {
		logrus.Errorf("error on request %+v", err)
		return "", errors.Wrap(err, "error performing token request")
	}
	defer transport.ResponseCloser(res)
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

func (s *Service) getGroups(host, token, realm string) ([]*group, error) {
	//https://keycloak-test1.apps.sedroche.openshiftworkshop.com/admin/realms/test1-john-test1/groups
	//https://keycloak-test1.apps.sedroche.openshiftworkshop.com/auth/admin/realms/test1-john-test1/groups?first=0&max=20
	//  %s/admin/realms/%s/groups
	u := fmt.Sprintf("%s/auth/admin/realms/%s/groups", host, realm)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer transport.ResponseCloser(res)
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected response code from listing groups " + res.Status)

	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var groups []*group
	if err := json.Unmarshal(data, &groups); err != nil {
		return nil, err
	}
	return groups, nil

}

func (s *Service) getUserID(host, token, realm, userName string) (string, error) {
	//	"/auth/admin/realms/%s/users?first=0&max=1&search=%s"
	u := fmt.Sprintf("%s/auth/admin/realms/%s/users?first=0&max=1&search=%s", host, realm, userName)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	res, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer transport.ResponseCloser(res)
	if res.StatusCode != http.StatusOK {
		return "", errors.New("unexpected response code from listing groups " + res.Status)

	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	logrus.Info("user found ", string(data))
	var users []*v1alpha1.User
	if err := json.Unmarshal(data, &users); err != nil {
		return "", err
	}
	if len(users) != 1 {
		logrus.Info("found users with length", len(users))
		return "", errors.New("failed to find user ")
	}
	return users[0].ID, nil
}

func (s *Service) addUserToGroups(host, token, realm, userID string, groups []string) error {
	//
	var errs string
	wg := &sync.WaitGroup{}
	for _, g := range groups {
		wg.Add(1)
		go func(group string) {
			defer wg.Done()
			req, err := http.NewRequest("PUT", fmt.Sprintf("%s/auth/admin/realms/%s/users/%s/groups/%s", host, realm, userID, group), nil)
			if err != nil {
				errs = errs + " " + err.Error()
				return
			}
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
			req.Header.Set("Content-Type", "application/json")
			res, err := s.httpClient.Do(req)
			if err != nil {
				errs = errs + " " + err.Error()
				return
			}
			defer transport.ResponseCloser(res)
			if res.StatusCode != http.StatusNoContent {
				errs = errs + " unexpected response code adding group for user" + res.Status
			}
		}(g)
	}
	wg.Wait()
	if errs != "" {
		return errors.New(errs)
	}
	return nil
}

func (s *Service) createUser(host, token, realm, userName, password string) (*v1alpha1.User, error) {
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
	}
	// may need to list available groups and add them after the user is created
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
	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request to create new enmasse user")
	}
	defer transport.ResponseCloser(res)
	if res.StatusCode == http.StatusConflict {
		logrus.Info("user already exists doing nothing")
		return nil, &errors2.AlreadyExistsErr{}

	}
	if res.StatusCode != http.StatusCreated {
		return nil, errors.New("failed to create user status code " + res.Status)
	}
	groups, err := s.getGroups(host, token, realm)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get keycloak groups")
	}

	var requiredGroups []string
	for _, g := range groups {
		if g.Name == "send_*" || g.Name == "recv_*" || g.Name == "manage" || g.Name == "view_*" || g.Name == "browse_*" {
			requiredGroups = append(requiredGroups, g.ID)
		}
	}
	id, err := s.getUserID(host, token, realm, userName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the userid after successful creation")
	}
	if err := s.addUserToGroups(host, token, realm, id, requiredGroups); err != nil {
		return nil, errors.Wrap(err, "failed to add required groups to user")
	}
	return &v1alpha1.User{UserName: userName, Password: password, ID: id}, nil
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

type group struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}
