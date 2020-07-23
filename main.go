package main

import (
	"fmt"
	"github.com/rancher/wrangler-cli"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/valyala/fasttemplate"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	autoAppsFlag                   = "autoapps"
	argoAPIVersion                 = "argoproj.io/v1alpha1"
	argoAppKind                    = "Application"
	autoAppsEnvPrefix              = "AUTOAPPS_"
	autoAppsAnnotationSkipDetector = "autoapps-skip-discovery"
)

// App is a bare bones struct barely descriptive enough to recognize ArgoCD Application CRDs
// NOTE: Purposely not using the argoproj types here, keep it simple!
type App struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Annotations map[string]string `yaml:"annotations"`
	}
}

type Generate struct {
	BasePath string `name:"basePath" usage:"Base path to begin traversal"`
}

func (g *Generate) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	if g.BasePath == "" {
		logrus.Fatal("You must specify a --basePath!")
	}

	apps, err := walkForApps(g.BasePath)
	if err != nil {
		logrus.Errorf("Failed to collect apps: %v", err)
	}

	// Remove any apps that don't want to be included

	// Print out rendered apps to stdout for ArgoCD to read
	fmt.Print(strings.Join(apps, "\n---\n"))

	return nil
}

func main() {
	root := cli.Command(&Generate{}, cobra.Command{
		Short: "Base path",
		Long:  "Base path long description",
	})
	cli.Main(root)
}

func walkForApps(base string) (apps []string, err error) {
	var appsData [][]byte
	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only care about valid yaml files
		if ext := filepath.Ext(path); ext == ".yaml" || ext == ".yml" {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			if ok := isAutoApp(data); ok {
				appsData = append(appsData, data)
			}
		}
		return nil
	})

	if err != nil {
		return apps, err
	}

	// Render apps template
	for _, data := range appsData {
		rendered := renderTemplate(string(data))

		// Determine if we need to skip it once it's read
		include := isAutoApp([]byte(rendered))

		if include {
			apps = append(apps, rendered)
		}
	}

	return apps, nil
}

// isAutoApp returns true/false based on whether or not a valid yaml file is a non skipped valid Application CR
func isAutoApp(data []byte) bool {
	var a App
	isApp := false

	err := yaml.Unmarshal(data, &a)
	if err != nil {}

	// Check if this is an app
	if a.ApiVersion == argoAPIVersion && a.Kind == argoAppKind {
		isApp = true
	}

	// Check if application is supposed to be skipped
	if val, ok := a.Metadata.Annotations[autoAppsAnnotationSkipDetector]; ok {
		if val == "true" {
			isApp = false
		}
	}

	return isApp
}

func renderTemplate(template string) string {
	t := fasttemplate.New(template, "{{", "}}")

	validEnvs := currentEnvToMap(autoAppsEnvPrefix)
	rendered := t.ExecuteString(validEnvs)
	return rendered
}

// currentEnvToMap will search for all environment variables with `prefix`, and convert their values into a map suitable for fasttemplate
func currentEnvToMap(prefix string) map[string]interface{} {
	envs := make(map[string]interface{})

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)

		// Trim detector prefix
		if strings.HasPrefix(pair[0], prefix) {
			trimmed := strings.TrimPrefix(pair[0], prefix)
			envs[trimmed] = pair[1]
		}

		// Always include ARGOCD_ variables
		if strings.HasPrefix(pair[0], "ARGOCD_") {
			envs[pair[0]] = pair[1]
		}
	}

	return envs
}
