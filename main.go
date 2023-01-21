package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/cmd/sops/formats"
	"go.mozilla.org/sops/v3/keyservice"

	"sigs.k8s.io/kustomize/api/loader"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// define the input API schema as a struct
type API struct {
	yaml.ResourceMeta

	Spec struct {
		// Replicas is the number of Deployment replicas
		// Defaults to the REPLICAS env var, or 1
		Files []string `yaml:"files,omitempty"`
	} `yaml:"spec,omitempty"`
}

const keepLocalConfigAnnotation = "krmfnsops.kaweezle.com/keep-local-config"

func Decrypt(b []byte, format formats.Format, file string, ignoreMac bool) (nodes []*yaml.RNode, err error) {

	store := common.StoreForFormat(format)

	// Load SOPS file and access the data key
	tree, err := store.LoadEncryptedFile(b)
	if err != nil {
		return nil, err
	}

	_, err = common.DecryptTree(common.DecryptTreeOpts{
		KeyServices: []keyservice.KeyServiceClient{
			keyservice.NewLocalClient(),
		},
		Tree:      &tree,
		IgnoreMac: ignoreMac,
		Cipher:    aes.NewCipher(),
	})

	if err != nil {
		return nil, err
	}

	var data []byte

	data, err = store.EmitPlainFile(tree.Branches)
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

func TransformNode(node *yaml.RNode) (result *yaml.RNode, err error) {
	var b []byte
	var nodes []*yaml.RNode

	ynode := node.YNode()
	b, err = yaml.Marshal(ynode)
	if err != nil {
		err = errors.Wrapf(err, "error reading manifest %q", ynode.Anchor)
		return
	}
	if nodes, err = Decrypt(b, formats.Yaml, ynode.Anchor, true); err != nil {
		err = errors.Wrapf(err, "error decoding manifest %q, content -->%s<--", ynode.Anchor, string(b))
		return
	}
	result = nodes[0]
	return
}

func main() {
	functionConfig := &API{}

	var filter kio.FilterFunc = func(incomingItems []*yaml.RNode) (items []*yaml.RNode, err error) {

		var b []byte
		var nodes []*yaml.RNode

		if strings.HasSuffix(functionConfig.Kind, "Transformer") {
			for _, node := range incomingItems {
				if node.Field("sops") != nil {
					node, err = TransformNode(node)
					if err != nil {
						return
					}
				}
				items = append(items, node)
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
				if nodes, err = Decrypt(b, format, file, false); err != nil {
					return
				}
				items = append(items, nodes...)
			}
		}

		return
	}

	// We replace the SimpleProcessor by a custom processor.
	// If the function config is encrypted (sops field), we assume this is an
	// _in place_ encrypted file. We decrypt and return the function config as a
	// resource. If not, we process normally, looking for files to decrypt or
	// acting as a transformer.
	var processor framework.ResourceListProcessorFunc = func(rl *framework.ResourceList) error {

		config := rl.FunctionConfig

		if config.Field("sops") != nil {
			item, err := TransformNode(config)
			if err != nil {
				return errors.Wrap(err, "transforming in place")
			}
			annotations := item.GetAnnotations()

			// We remove the function annotation
			delete(annotations, "config.kubernetes.io/function")

			// We may want to keep the local-config annotation in order to have
			// The resource kept out of the output. The use case is make
			// available secrets for replacement and then forget them.
			_, keepLocalConfig := annotations[keepLocalConfigAnnotation]
			if keepLocalConfig {
				delete(annotations, keepLocalConfigAnnotation)
			} else {
				delete(annotations, filters.LocalConfigAnnotation)
			}
			item.SetAnnotations(annotations)
			//b, _ := yaml.Marshal(items[0].YNode())
			//return fmt.Errorf("item:\n%s", string(b))
			rl.Items = []*yaml.RNode{item}
		} else {
			if err := framework.LoadFunctionConfig(rl.FunctionConfig, functionConfig); err != nil {
				return errors.Wrap(err, "loading function config")
			}
			return errors.Wrap(rl.Filter(filter), "processing filter")
		}

		return nil
	}

	cmd := command.Build(processor, command.StandaloneDisabled, false)
	command.AddGenerateDockerfile(cmd)
	cmd.Version = "v0.1.5" // <---VERSION--->

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
