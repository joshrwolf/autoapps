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
	"github.com/drone/envsubst"
	"strings"
)

const (
	autoAppsFlag = "autoapps"
)

type Generate struct {
	BasePath string `name:"basePath" usage:"Base path to begin traversal"`
}

func (g *Generate) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	fmt.Printf("Your option one %s and args %v\n", g.BasePath, args)

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

func walkForApps(base string) (apps []string, err error) {
	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only care about valid yaml files
		if ext := filepath.Ext(path); ext == ".yaml" || ext == ".yml" {
			var a MiniApp
			dat, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			err = yaml.Unmarshal(dat, &a)
			if err != nil {
				logrus.Warnf("Failed to unmarshal %s: %v", path, err)
			}

			if a.ApiVersion == "argoproj.io/v1alpha1" && a.Kind == "Application" {
				if _, ok := a.Metadata.Annotations[autoAppsFlag]; ok {
					apps = append(apps, safeEnvSubst(string(dat)))
				}
			}
		}
		return nil
	})

	if err != nil {
		return apps, err
	}

	return apps, nil
}

// TODO: Need to implement a way to make this "safe" and only support _allowed_ environment variables
//		 Make it obvious how "allowed" envs are determined
func safeEnvSubst(original string) string {
	substituted, err := envsubst.EvalEnv(original)
	if err != nil {
		logrus.Fatalf("Failed to substitute: %v", err)
	}

	return substituted
}