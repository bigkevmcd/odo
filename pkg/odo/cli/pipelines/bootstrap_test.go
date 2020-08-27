package pipelines

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"testing"

	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/pipelines"
	"github.com/spf13/cobra"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
)

type mockSpinner struct {
	writer io.Writer
	start  bool
	end    bool
}
type keyValuePair struct {
	key   string
	value string
}

func TestCompleteBootstrapParameters(t *testing.T) {
	completeTests := []struct {
		name        string
		prefix      string
		wantPrefix  string
		serviceRepo string
		gitRepo     string
	}{
		{"no prefix", "", "", "https://github.com/gaganhegde/test-repo.git", "https://github.com/gaganhegde/taxi.git"},
		{"prefix with hyphen", "test-", "test-", "https://github.com/gaganhegde/test-repo.git", "https://github.com/gaganhegde/taxi.git"},
		{"prefix without hyphen", "test", "test-", "https://github.com/gaganhegde/test-repo.git", "https://github.com/gaganhegde/taxi.git"},
	}

	for _, tt := range completeTests {
		o := BootstrapParameters{
			&pipelines.BootstrapOptions{Prefix: tt.prefix, GitOpsRepoURL: tt.gitRepo, ServiceRepoURL: tt.serviceRepo, ImageRepo: ""},
			&genericclioptions.Context{},
		}

		err := o.Validate()

		if err != nil {
			t.Errorf("Complete() %#v failed: ", err)
		}

		if o.Prefix != tt.wantPrefix {
			t.Errorf("Complete() %#v prefix: got %s, want %s", tt.name, o.Prefix, tt.wantPrefix)
		}
	}
}

func TestAddSuffixWithBootstrap(t *testing.T) {
	gitOpsURL := "https://github.com/org/gitops"
	appURL := "https://github.com/org/app"
	tt := []struct {
		name           string
		gitOpsURL      string
		appURL         string
		validGitOpsURL string
		validAppURL    string
	}{
		{"suffix already exists", gitOpsURL + ".git", appURL + ".git", gitOpsURL + ".git", appURL + ".git"},
		{"misssing suffix", gitOpsURL, appURL, gitOpsURL + ".git", appURL + ".git"},
	}

	for _, test := range tt {
		t.Run(test.name, func(rt *testing.T) {
			o := BootstrapParameters{
				&pipelines.BootstrapOptions{
					GitOpsRepoURL:  test.gitOpsURL,
					ServiceRepoURL: test.appURL},
				&genericclioptions.Context{}}

			err := o.Validate()
			if err != nil {
				t.Errorf("Complete() %#v failed: ", err)
			}

			if o.GitOpsRepoURL != test.validGitOpsURL {
				rt.Fatalf("URL mismatch: got %s, want %s", o.GitOpsRepoURL, test.validAppURL)
			}
			if o.ServiceRepoURL != test.validAppURL {
				rt.Fatalf("URL mismatch: got %s, want %s", o.GitOpsRepoURL, test.validAppURL)
			}
		})
	}
}

func TestValidateBootstrapParameters(t *testing.T) {
	optionTests := []struct {
		name    string
		gitRepo string
		errMsg  string
	}{
		{"invalid repo", "test", "repo must be org/repo"},
		{"valid repo", "test/repo", ""},
	}

	for _, tt := range optionTests {
		o := BootstrapParameters{
			&pipelines.BootstrapOptions{
				GitOpsRepoURL: tt.gitRepo,
				Prefix:        "test"},
			&genericclioptions.Context{},
		}
		err := o.Validate()

		if err != nil && tt.errMsg == "" {
			t.Errorf("Validate() %#v got an unexpected error: %s", tt.name, err)
			continue
		}

		if !matchError(t, tt.errMsg, err) {
			t.Errorf("Validate() %#v failed to match error: got %s, want %s", tt.name, err, tt.errMsg)
		}
	}
}

// func TestBootstrapCommandWithMissingParams(t *testing.T) {
// 	cmdTests := []struct {
// 		desc    string
// 		flags   []keyValuePair
// 		wantErr string
// 	}{
// 		{"Missing gitops-repo-url flag",
// 			[]keyValuePair{flag("service-repo-url", "sample/repo"), flag("image-repo", "registry/username/repo")},
// 			`required flag(s) "gitops-repo-url" not set`},
// 		{"Missing service-repo-url flag",
// 			[]keyValuePair{
// 				flag("gitops-repo-url", "sample/repo"), flag("image-repo", "registry/username/repo")},
// 			`required flag(s) "service-repo-url" not set`},
// 		{"Missing image-repo flag",
// 			[]keyValuePair{
// 				flag("gitops-repo-url", "sample/repo"), flag("service-repo-url", "sample/repo")},
// 			`required flag(s) "image-repo" not set`},
// 	}
// 	for _, tt := range cmdTests {
// 		t.Run(tt.desc, func(t *testing.T) {
// 			_, _, err := executeCommand(NewCmdBootstrap("bootstrap", "odo pipelines bootstrap"), tt.flags...)
// 			if err.Error() != tt.wantErr {
// 				t.Errorf("got %s, want %s", err, tt.wantErr)
// 			}
// 		})
// 	}
// }

//
func executeCommand(cmd *cobra.Command, flags ...keyValuePair) (c *cobra.Command, output string, err error) {
	fmt.Println("THis is cmd", cmd)
	buf := new(bytes.Buffer)
	cmd.SetOutput(buf)
	for _, flag := range flags {
		cmd.Flags().Set(flag.key, flag.value)
	}
	c, err = cmd.ExecuteC()
	return c, buf.String(), err
}

func matchError(t *testing.T, s string, e error) bool {
	t.Helper()
	if s == "" && e == nil {
		return true
	}
	if s != "" && e == nil {
		return false
	}
	match, err := regexp.MatchString(s, e.Error())
	if err != nil {
		t.Fatal(err)
	}
	return match
}

func flag(k, v string) keyValuePair {
	return keyValuePair{
		key:   k,
		value: v,
	}
}

func (m *mockSpinner) Start(status string, debug bool) {
	m.start = true
	fmt.Fprintf(m.writer, "\n%s", status)
}

func (m *mockSpinner) End(status bool) {
	m.end = status
}

func (m *mockSpinner) WarningStatus(status string) {
	fmt.Fprintf(m.writer, "[%s]", status)
}

func TestCheckSpinner(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		endStatus bool
		wantMsg   string
	}{
		{
			"No error",
			nil,
			true,
			"\nChecking if abcd is installed",
		},
		{
			"Resource not found error",
			errors.NewNotFound(schema.GroupResource{}, "abcd"),
			false,
			"\nChecking if abcd is installed[Please install abcd]",
		},
		{
			"Random cluster error",
			fmt.Errorf("Sample cluster error"),
			false,
			"\nChecking if abcd is installed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buff := &bytes.Buffer{}

			fakeSpinner := &mockSpinner{writer: buff}
			fakeSpinner.Start("Checking if abcd is installed", false)
			setSpinnerStatus(fakeSpinner, "Please install abcd", test.err)

			if fakeSpinner.end != test.endStatus {
				t.Errorf("Spinner status mismatch: got %v, want %v", fakeSpinner.end, test.endStatus)
			}
			assertMessage(t, buff.String(), test.wantMsg)
		})
	}
}

func TestDependenciesWithNothingInstalled(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	wantMsg := `
Checking if Sealed Secrets is installed with the default configuration[Please install Sealed Secrets from https://github.com/bitnami-labs/sealed-secrets/releases]
Checking if ArgoCD Operator is installed with the default configuration[Please install ArgoCD operator from OperatorHub]
Checking if OpenShift Pipelines Operator is installed with the default configuration[Please install OpenShift Pipelines operator from OperatorHub]`

	buff := &bytes.Buffer{}
	fakeSpinner := &mockSpinner{writer: buff}
	err := checkBootstrapDependencies(&BootstrapParameters{&pipelines.BootstrapOptions{}, &genericclioptions.Context{}}, fakeClient, fakeSpinner)
	wantErr := "Failed to satisfy the required dependencies"

	assertError(t, err, wantErr)
	assertMessage(t, buff.String(), wantMsg)
}

func TestDependenciesWithAllInstalled(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(sealedSecretService(), argoCDOperator(), pipelinesOperator())

	wantMsg := `
Checking if Sealed Secrets is installed with the default configuration
Checking if ArgoCD Operator is installed with the default configuration
Checking if OpenShift Pipelines Operator is installed with the default configuration`

	buff := &bytes.Buffer{}
	fakeSpinner := &mockSpinner{writer: buff}
	wizardParams := &BootstrapParameters{&pipelines.BootstrapOptions{}, &genericclioptions.Context{}}
	err := checkBootstrapDependencies(wizardParams, fakeClient, fakeSpinner)

	assertError(t, err, "")
	if wizardParams.SealedSecretsService.Name != "sealed-secrets-controller" && wizardParams.SealedSecretsService.Namespace != "kube-system" {
		t.Fatalf("Expected sealed secrets to be set")
	}
	assertMessage(t, buff.String(), wantMsg)
}

func TestDependenciesWithNoArgoCD(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(sealedSecretService(), pipelinesOperator())

	wantMsg := `
Checking if Sealed Secrets is installed with the default configuration
Checking if ArgoCD Operator is installed with the default configuration[Please install ArgoCD operator from OperatorHub]
Checking if OpenShift Pipelines Operator is installed with the default configuration`

	buff := &bytes.Buffer{}
	fakeSpinner := &mockSpinner{writer: buff}
	wizardParams := &BootstrapParameters{&pipelines.BootstrapOptions{}, &genericclioptions.Context{}}
	err := checkBootstrapDependencies(wizardParams, fakeClient, fakeSpinner)
	wantErr := "Failed to satisfy the required dependencies"

	assertError(t, err, wantErr)
	assertMessage(t, buff.String(), wantMsg)
}

func TestDependenciesWithNoPipelines(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(sealedSecretService(), argoCDOperator())

	wantMsg := `
Checking if Sealed Secrets is installed with the default configuration
Checking if ArgoCD Operator is installed with the default configuration
Checking if OpenShift Pipelines Operator is installed with the default configuration[Please install OpenShift Pipelines operator from OperatorHub]`

	buff := &bytes.Buffer{}
	fakeSpinner := &mockSpinner{writer: buff}
	wizardParams := &BootstrapParameters{&pipelines.BootstrapOptions{}, &genericclioptions.Context{}}
	err := checkBootstrapDependencies(wizardParams, fakeClient, fakeSpinner)
	wantErr := "Failed to satisfy the required dependencies"

	assertError(t, err, wantErr)
	assertMessage(t, buff.String(), wantMsg)
}

func assertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		if msg != "" {
			t.Fatalf("Error mismatch: got %v, want %v", err, msg)
		}
		return
	}
	if err.Error() != msg {
		t.Fatalf("Error mismatch: got %s, want %s", err.Error(), msg)
	}
}

func assertMessage(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("Message mismatch: got %s, want %s", got, want)
	}
}

func sealedSecretService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sealed-secrets-controller",
			Namespace: "kube-system",
		},
	}
}

func argoCDOperator() *appv1.DeploymentList {
	return &appv1.DeploymentList{
		Items: []appv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "argocd-operator",
					Namespace: "argocd",
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name:      "argocd-server",
					Namespace: "argocd",
				},
			},
		},
	}
}

func pipelinesOperator() *appv1.Deployment {
	return &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openshift-pipelines-operator",
			Namespace: "openshift-operators",
		},
	}
}
