package main

import (
	"fmt"
	"github.com/rancher/wrangler-cli"
	"gopkg.in/yaml.v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/a8m/envsubst"
	"strings"
)

const (
	autoAppsFlag = "autoapps"
	autoAppsAnnotationSkipVal = "skip"
	argoAPIVersion = "argoproj.io/v1alpha1"
	argoAppKind = "Application"
)

type Generate struct {
	BasePath string `name:"basePath" usage:"Base path to begin traversal"`
}

func (g *Generate) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	if g.BasePath != "" {
		apps, err := walkForApps(g.BasePath)
		if err != nil {
			logrus.Errorf("Failed to collect apps: %v", err)
		}

		// Print out rendered apps to stdout for ArgoCD to read
		fmt.Print(strings.Join(apps, "\n---\n"))
	}

	return nil
}

func main() {
	root := cli.Command(&Generate{}, cobra.Command{
		Short: "Base path",
		Long: "Base path long description",
	})
	cli.Main(root)
}

// MiniApp is a bare bones struct barely descriptive enough to recognize ArgoCD Application CRDs
// NOTE: Purposely not using the argoproj types here, keep it simple!
// TODO: Only recognize apps annotated a certain way
type MiniApp struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind string `yaml:"kind"`
	Metadata struct {
		Annotations map[string]string `yaml:"annotations"`
	}
}

type App struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind string `yaml:"kind"`
	Metadata struct {
		Annotations map[string]string `yaml:"annotations"`
	}
}

type Metadata struct {
	Annotations map[string]string `yaml:"annotations"`
}

func walkForApps(base string) (apps []string, err error) {
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

			// TODO: This whole thing is some laaaazy logic flow
			isApp, _ := isApp(data)
			if !isApp {
				// Bailout if it's not an app, we don't care anymore
				return nil
			}

			// Render envsubst
			render, substitutedApp := safeEnvSubst(data)
			if render {
				apps = append(apps, substitutedApp)
			}
		}
		return nil
	})

	if err != nil {
		return apps, err
	}

	return apps, nil
}

func isApp(data []byte) (bool, App) {
	var a App
	isApp := false

	err := yaml.Unmarshal(data, &a)
	if err != nil {}

	// Check if this is an app
	if a.ApiVersion == argoAPIVersion && a.Kind == argoAppKind {
		isApp = true
	}

	return isApp, a
}

func isAutoApp(data []byte) (bool, MiniApp) {
	var m MiniApp

	err := yaml.Unmarshal(data, &m)
	// Gobble up errors, need to keep stdout clean and stderr empty
	if err != nil {}

	// Check if this is an autoapp
	if m.ApiVersion == argoAPIVersion && m.Kind	== argoAppKind {
		if _, ok := m.Metadata.Annotations[autoAppsFlag]; ok {
			return true, m
		}
	}

	return false, m
}

// TODO: Need to implement a way to make this "safe" and only support _allowed_ environment variables
//		 Make it obvious how "allowed" envs are determined
func safeEnvSubst(original []byte) (bool, string) {
	render := false

	substituted, err := envsubst.Bytes(original)
	if err != nil {
		logrus.Fatalf("Failed to substitute: %v", err)
	}

	// Only return valid yaml if annotations trigger is true
	var a App
	err = yaml.Unmarshal(substituted, &a)
	if err != nil {}

	if val, ok := a.Metadata.Annotations[autoAppsFlag]; ok {
		if val != autoAppsAnnotationSkipVal {
			render = true
			return render, string(substituted)
		}
	}

	return render, ""
}