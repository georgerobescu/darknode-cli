package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/renproject/darknode-cli/darknode"
	"github.com/renproject/darknode-cli/util"
	"github.com/urfave/cli"
)

// ErrUnknownProvider is returned when user tries to deploy a darknode with an unknown cloud provider.
var ErrUnknownProvider = errors.New("unknown cloud provider")

// ErrUnsupportedInstanceType is returned when the selected instance type cannot be created to user account.
var ErrInstanceTypeNotAvailable = errors.New("selected instance type is not available")

// ErrRegionNotAvailable is returned when the selected region is not available to user account.
var ErrRegionNotAvailable = errors.New("selected region is not available")

var (
	NameAws = "aws"
	NameDo  = "do"
	NameGcp = "gcp"
)

type Provider interface {
	Name() string
	Deploy(ctx *cli.Context) error
}

func ParseProvider(ctx *cli.Context) (Provider, error) {
	if ctx.Bool(NameAws) {
		return NewAws(ctx)
	}

	if ctx.Bool(NameDo) {
		return NewDo(ctx)
	}

	// if ctx.Bool(NameGcp) {
	// 	return
	// }

	return nil, ErrUnknownProvider
}

// Provider returns the provider of a darknode instance.
func GetProvider(name string) (string, error) {
	if name == "" {
		return "", util.ErrEmptyName
	}

	cmd := fmt.Sprintf("cd %v && terraform output provider", util.NodePath(name))
	provider, err := util.CommandOutput(cmd)
	return strings.TrimSpace(provider), err
}

// initialise all files needed by deploying a new node
func initNode(name, tags string, network darknode.Network) error {
	if err := util.InitNodeDirectory(name, tags); err != nil {
		return err
	}
	if err := util.GenerateSshKeyAndWriteToDir(name); err != nil {
		return err
	}

	// Generate a new config and write to a file.
	config, err := darknode.NewConfig(network)
	if err != nil {
		return err
	}
	configData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	configPath := filepath.Join(util.NodePath(name), "config.json")
	return ioutil.WriteFile(configPath, configData, 0600)
}

func runTerraform(name string) error {
	path := util.NodePath(name)
	init := fmt.Sprintf("cd %v && terraform init", path)
	if err := util.Run("bash", "-c", init); err != nil {
		return err
	}

	fmt.Println("Deploying darknode ... ")
	apply := fmt.Sprintf("cd %v && terraform apply -auto-approve -no-color", path)
	return util.Run("bash", "-c", apply)
}

// outputURL writes success message and the URL for registering the node to the terminal.
func outputURL(name string) error {
	url, err := util.RegisterUrl(name)
	if err != nil {
		return err
	}
	color.Green("")
	color.Green("Congratulations! Your Darknode is deployed.")
	color.Green("Join the network by registering your Darknode at %s", url)
	return util.OpenInBrowser(url)
}

func ipfsUrl(network darknode.Network) string {
	switch network {
	case darknode.Mainnet:
		panic("unsupported")
	case darknode.Chaosnet:
		return "http://157.245.76.68:8080/ipns/QmVq3uLmSpxQoz7Zk7RBaeiBb1DVaKVcCSkPGZKG9xbqvy"
	case darknode.Testnet:
		return "http://178.128.49.72:8080/ipns/QmU955UGWJFbnEJZMHszhWTP9YBiaxqs2g4Hiw2AP3jXwn"
	case darknode.Devnet:
		return "http://178.128.49.72:8080/ipns/Qma5uQ7HL87FbuDQZhvQQzc4wyoXY4P7YfKRSCoBy6qgFv"
	default:
		panic("unknown network")
	}
}
