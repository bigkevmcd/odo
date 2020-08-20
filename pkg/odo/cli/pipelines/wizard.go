package pipelines

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/openshift/odo/pkg/log"
	"github.com/openshift/odo/pkg/odo/cli/pipelines/ui"
	"github.com/openshift/odo/pkg/odo/cli/pipelines/utility"
	"github.com/openshift/odo/pkg/odo/genericclioptions"
	"github.com/openshift/odo/pkg/pipelines"
	"github.com/openshift/odo/pkg/pipelines/ioutils"
	"github.com/openshift/odo/pkg/pipelines/namespaces"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const (
	// WizardRecommendedCommandName the recommended command name
	WizardRecommendedCommandName = "wizard"

	sealedSecretsName   = "sealed-secrets-controller"
	sealedSecretsNS     = "kube-system"
	argoCDNS            = "argocd"
	argoCDOperatorName  = "argocd-operator"
	argoCDServerName    = "argocd-server"
	pipelinesOperatorNS = "openshift-operators"
)

var (
	WizardExample = ktemplates.Examples(`
    # Wizard OpenShift pipelines.
    %[1]s 
    `)

	WizardLongDesc  = ktemplates.LongDesc(`Wizard GitOps CI/CD Manifest`)
	WizardShortDesc = `Wizard pipelines with a starter configuration`
)

// WizardParameters encapsulates the parameters for the odo pipelines init command.
type WizardParameters struct {
	*pipelines.BootstrapOptions
	// generic context options common to all commands
	*genericclioptions.Context
}

// NewWizardParameters Wizards a WizardParameters instance.
func NewWizardParameters() *WizardParameters {
	return &WizardParameters{
		BootstrapOptions: &pipelines.BootstrapOptions{},
	}
}

// Complete completes WizardParameters after they've been created.
//
// If the prefix provided doesn't have a "-" then one is added, this makes the
// generated environment names nicer to read.
func (io *WizardParameters) Complete(name string, cmd *cobra.Command, args []string) error {

	clientSet, err := namespaces.GetClientSet()
	if err != nil {
		return err
	}
	err = checkBootstrapDependencies(io, clientSet)
	if err != nil {
		return err
	}

	// ask for sealed secrets only when default is absent
	if io.SealedSecretsService == (types.NamespacedName{}) {
		io.SealedSecretsService.Name = ui.EnterSealedSecretService()
		io.SealedSecretsService.Namespace = ui.EnterSealedSecretNamespace()
	}

	io.GitOpsRepoURL = ui.EnterGitRepo()
	option := ui.SelectOptionImageRepository()
	if option == "Openshift Internal repository" {
		io.InternalRegistryHostname = ui.EnterInternalRegistry()
		io.ImageRepo = ui.EnterImageRepoInternalRegistry()

	} else {
		io.DockerConfigJSONFilename = ui.EnterDockercfg()
		io.ImageRepo = ui.EnterImageRepoExternalRepository()
	}
	io.GitOpsWebhookSecret = ui.EnterGitWebhookSecret()
	commitStatusTrackerCheck := ui.SelectOptionCommitStatusTracker()
	if commitStatusTrackerCheck == "yes" {
		io.StatusTrackerAccessToken = ui.EnterStatusTrackerAccessToken()
	}
	io.Prefix = ui.EnterPrefix()
	io.Prefix = utility.MaybeCompletePrefix(io.Prefix)
	io.ServiceRepoURL = ui.EnterServiceRepoURL()
	if io.ServiceRepoURL != "" {
		io.ServiceWebhookSecret = ui.EnterServiceWebhookSecret()
		io.ServiceRepoURL = utility.AddGitSuffixIfNecessary(io.ServiceRepoURL)
	}

	io.OutputPath = ui.EnterOutputPath(io.GitOpsRepoURL)
	exists, _ := ioutils.IsExisting(ioutils.NewFilesystem(), filepath.Join(io.OutputPath, "pipelines.yaml"))
	if exists {
		selectOverwriteOption := ui.SelectOptionOverwrite()
		if selectOverwriteOption == "no" {
			io.Overwrite = false
			return fmt.Errorf("Cannot create GitOps configuration since file exists at %s", io.OutputPath)
		}
	}
	io.Overwrite = true
	io.GitOpsRepoURL = utility.AddGitSuffixIfNecessary(io.GitOpsRepoURL)
	return nil
}

func checkBootstrapDependencies(io *WizardParameters, kubeClient kubernetes.Interface) error {

	client := utility.NewClient(kubeClient)
	log.Progressf("\nChecking dependencies\n")

	sealedSpinner := log.Spinner("Checking if Sealed Secrets is installed at kube-system namespace")
	err := client.CheckIfSealedSecretsExists(sealedSecretsNS+"s", sealedSecretsName)
	if err != nil {
		sealedSpinner.WarningStatus("Please install Sealed Secrets from https://github.com/bitnami-labs/sealed-secrets/releases")
		sealedSpinner.End(false)
	} else {
		io.SealedSecretsService.Name = sealedSecretsName
		io.SealedSecretsService.Namespace = sealedSecretsNS
		sealedSpinner.End(true)
	}

	argoSpinner := log.Spinner("Checking if ArgoCD Operator is installed at argocd namespace")
	err = client.CheckIfArgoCDExists(argoCDNS)
	if err != nil {
		argoSpinner.WarningStatus("Please install ArgoCD operator from OperatorHub")
		argoSpinner.End(false)
	} else {
		argoSpinner.End(true)
	}

	pipelineSpinner := log.Spinner("Checking if OpenShift Pipelines Operator is installed")
	err = client.CheckIfPipelinesExists(pipelinesOperatorNS)
	if err != nil {
		pipelineSpinner.WarningStatus("Please install OpenShift Pipelines operator from OperatorHub")
		pipelineSpinner.End(false)
	} else {
		pipelineSpinner.End(true)
	}

	if err != nil {
		return fmt.Errorf("Failed to satisfy the required dependencies")
	}
	return nil
}

// Validate validates the parameters of the WizardParameters.
func (io *WizardParameters) Validate() error {
	gr, err := url.Parse(io.GitOpsRepoURL)
	if err != nil {
		return fmt.Errorf("failed to parse url %s: %w", io.GitOpsRepoURL, err)
	}

	// TODO: this won't work with GitLab as the repo can have more path elements.
	if len(utility.RemoveEmptyStrings(strings.Split(gr.Path, "/"))) != 2 {
		return fmt.Errorf("repo must be org/repo: %s", strings.Trim(gr.Path, ".git"))
	}

	return nil
}

// Run runs the project Wizard command.
func (io *WizardParameters) Run() error {
	if io.ServiceRepoURL != "" {
		err := pipelines.Bootstrap(io.BootstrapOptions, ioutils.NewFilesystem())
		if err != nil {
			return err
		}
		log.Success("Bootstrapped GitOps sucessfully.")
	}
	return nil
}

// NewCmdWizard creates the project init command.
func NewCmdWizard(name, fullName string) *cobra.Command {
	o := NewWizardParameters()

	wizardCmd := &cobra.Command{
		Use:     name,
		Short:   WizardShortDesc,
		Long:    WizardLongDesc,
		Example: fmt.Sprintf(WizardExample, fullName),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}
	return wizardCmd
}
