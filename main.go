package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/decrypt"

	"sigs.k8s.io/kustomize/api/loader"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// define the input API schema as a struct
type API struct {
	Kind       string `yaml:"name"`
	ApiVersion string `yaml:"apiVersion"`
	Metadata   struct {
		// Name is the Deployment Resource and Container name
		Name string `yaml:"name"`
	} `yaml:"metadata"`

	Spec struct {
		// Replicas is the number of Deployment replicas
		// Defaults to the REPLICAS env var, or 1
		Files []string `yaml:"files"`
	} `yaml:"spec"`
}

func Decrypt(b []byte, format formats.Format, file string) (nodes []*yaml.RNode, err error) {
	var data []byte
	data, err = decrypt.DataWithFormat(b, format)
	if err != nil {
		err = errors.Wrapf(err, "trouble decrypting file %s", file)
		return
	}

	nodes, err = kio.FromBytes(data)
	if err != nil {
		err = errors.Wrapf(err, "Error while reading decrypted resources from file %s", file)
	}
	return
}

func main() {
	functionConfig := &API{}

	var p kio.FilterFunc = func(incomingItems []*yaml.RNode) (items []*yaml.RNode, err error) {

		var b []byte
		var nodes []*yaml.RNode

		if strings.HasSuffix(functionConfig.Kind, "Transformer") {
			for _, node := range incomingItems {
				if node.Field("sops") != nil {

					ynode := node.YNode()
					b, err = yaml.Marshal(ynode)
					if err != nil {
						err = errors.Wrapf(err, "error reading manifest %q", ynode.Anchor)
						return
					}
					if nodes, err = Decrypt(b, formats.Yaml, ynode.Anchor); err != nil {
						return
					}
					items = append(items, nodes...)

				} else {
					items = append(items, node)
				}
			}

		} else {
			if functionConfig.Spec.Files == nil {
				err = fmt.Errorf("generator configuration doesn't contain any file")
				return
			}

			// This loader allows loading from URLs
			file_loader := loader.NewFileLoaderAtCwd(filesys.MakeFsOnDisk())

			for _, file := range functionConfig.Spec.Files {

				b, err = file_loader.Load(file)
				if err != nil {
					err = errors.Wrapf(err, "error reading manifest %q", file)
					return
				}

				format := formats.FormatForPath(file)
				if nodes, err = Decrypt(b, format, file); err != nil {
					return
				}
				items = append(items, nodes...)
			}
		}

		return
	}

	cmd := command.Build(framework.SimpleProcessor{Filter: p, Config: functionConfig}, command.StandaloneDisabled, false)
	command.AddGenerateDockerfile(cmd)
	cmd.Version = "v0.1.0" // <---VERSION--->

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
