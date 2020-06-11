package imagerepo

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/openshift/odo/pkg/pipelines/meta"
	"github.com/openshift/odo/pkg/pipelines/namespaces"
	"github.com/openshift/odo/pkg/pipelines/roles"

	"github.com/openshift/odo/pkg/pipelines/config"

	res "github.com/openshift/odo/pkg/pipelines/resources"

	corev1 "k8s.io/api/core/v1"
)

// ValidateImageRepo validates the input image repo.  It determines if it is
// for internal registry and prepend internal registry hostname if neccessary.
func ValidateImageRepo(imageRepo, registryURL string) (bool, string, error) {
	components := strings.Split(imageRepo, "/")

	// repo url has minimum of 2 components
	if len(components) < 2 {
		return false, "", imageRepoValidationErrors(imageRepo)
	}

	for _, v := range components {
		if isBlank(v) {
			return false, "", imageRepoValidationErrors(imageRepo)
		}
	}

	if len(components) == 2 {
		if components[0] == "docker.io" || components[0] == "quay.io" {
			// we recognize docker.io and quay.io.  It is missing one component
			return false, "", imageRepoValidationErrors(imageRepo)
		}
		// We have format like <project>/<app> which is an internal registry.
		// We prepend the internal registry hostname.
		return true, registryURL + "/" + imageRepo, nil
	}

	// Check the first component to see if it is an internal registry
	if len(components) == 3 {
		return components[0] == registryURL, imageRepo, nil
	}

	// > 3 components.  invalid repo
	return false, "", imageRepoValidationErrors(imageRepo)
}

func isBlank(s string) bool {
	return strings.TrimSpace(s) == "" || len(s) > len(strings.TrimSpace(s))
}

func imageRepoValidationErrors(imageRepo string) error {
	return fmt.Errorf("failed to parse image repo:%s, expected image repository in the form <registry>/<username>/<repository> or <project>/<app> for internal registry", imageRepo)
}

<<<<<<< HEAD
=======
// CreateInternalRegistryResources creates the resources for accessing the
// internal registry.
>>>>>>> 2de59d82a9c6d8dd41231e5bc4b88d18c9e7dd10
func CreateInternalRegistryResources(cfg *config.PipelinesConfig, sa *corev1.ServiceAccount, imageRepo string) (res.Resources, error) {
	// Provide access to service account for using internal registry
	namespace := strings.Split(imageRepo, "/")[1]

	resources := res.Resources{}
	filename := filepath.Join("01-namespaces", fmt.Sprintf("%s.yaml", namespace))
<<<<<<< HEAD
	namespacePath := filepath.Join(config.PathForPipelines(cfg), "base", "pipelines", filename)
	resources[namespacePath] = namespaces.Create(namespace)
	filenames = append(filenames, filename)

	filename, roleBinding := createInternalRegistryRoleBinding(cfg, namespace, sa)
=======
	resources[filename] = namespaces.Create(namespace)
	roleBinding := createInternalRegistryRoleBinding(cfg, namespace, sa)
>>>>>>> 2de59d82a9c6d8dd41231e5bc4b88d18c9e7dd10
	return res.Merge(roleBinding, resources), nil
}

func createInternalRegistryRoleBinding(cfg *config.PipelinesConfig, ns string, sa *corev1.ServiceAccount) res.Resources {
	roleBindingName := fmt.Sprintf("internal-registry-%s-binding", ns)
	roleBindingFilename := filepath.Join("02-rolebindings", fmt.Sprintf("%s.yaml", roleBindingName))
	return res.Resources{roleBindingFilename: roles.CreateRoleBinding(meta.NamespacedName(ns, roleBindingName), sa, "ClusterRole", "edit")}
}
