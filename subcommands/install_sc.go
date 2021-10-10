package subcommands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/client"
	"github.com/everettraven/packageless/utils"
)

//Install Sub-Command Object
type InstallCommand struct {
	//FlagSet so that we can create a custom flag
	fs *flag.FlagSet

	//String for the name of the pim to install
	name string

	//Tools that can be used by the command
	tools utils.Tools

	cp utils.Copier

	config utils.Config
}

//Instantiation method for a new InstallCommand
func NewInstallCommand(tools utils.Tools, cp utils.Copier, config utils.Config) *InstallCommand {
	//Create a new InstallCommand and set the FlagSet
	ic := &InstallCommand{
		fs:     flag.NewFlagSet("install", flag.ContinueOnError),
		tools:  tools,
		cp:     cp,
		config: config,
	}

	return ic
}

//Name - Gets the name of the Sub-Command
func (ic *InstallCommand) Name() string {
	return ic.fs.Name()
}

//Init - Parses and Populates values of the Install subcommand
func (ic *InstallCommand) Init(args []string) error {

	if len(args) <= 0 {
		return errors.New("No pim name was found. You must include the name of the pim you wish to install.")
	}

	ic.name = args[0]
	return nil
}

//Run - Runs the install subcommand
func (ic *InstallCommand) Run() error {
	//Create variables to use later
	var found bool
	var pim utils.PackageImage
	var version utils.Version

	var pimName string
	var pimVersion string

	if strings.Contains(ic.name, ":") {
		split := strings.Split(ic.name, ":")
		pimName = split[0]
		pimVersion = split[1]
	} else {
		pimName = ic.name
		pimVersion = "latest"
	}

	//Create the Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	//Create a variable for the executable directory
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	ed := filepath.Dir(ex)

	//Default location of the pim list
	pimList := ed + "/package_list.hcl"

	pimListBody, err := ic.tools.GetHCLBody(pimList)

	if err != nil {
		return err
	}

	//Parse the pim list
	parseOut, err := ic.tools.ParseBody(pimListBody, utils.PimHCLUtil{})

	pims := parseOut.(utils.PimHCLUtil)

	//Check for errors
	if err != nil {
		return err
	}

	//Look for the pim we want in the pim list
	for _, pimItem := range pims.Pims {
		//If we find it, set some variables and break
		if pimItem.Name == pimName {
			pim = pimItem

			for _, ver := range pim.Versions {
				if ver.Version == pimVersion {
					found = true
					version = ver
					break
				}
			}
		}
	}

	//Make sure we have found the pim in the pim list
	if !found {
		return errors.New("Could not find pim " + pimName + " with version '" + pimVersion + "' in the pim list")
	}

	//Check if the corresponding pim image is already installed
	imgExist, err := ic.tools.ImageExists(version.Image, cli)

	//Check for errors
	if err != nil {
		return err
	}

	//If the image exists the pim is already installed
	if imgExist {
		return errors.New("pim " + pim.Name + " is already installed")
	}

	fmt.Println("Installing", pim.Name+":"+version.Version)
	//Pull the image down from Docker Hub
	err = ic.tools.PullImage(version.Image, cli)

	if err != nil {
		return err
	}

	fmt.Println("Creating pim directories")

	//Create the base directory for the pim
	err = ic.tools.MakeDir(ed + pim.BaseDir)

	if err != nil {
		return err
	}

	//Check the volumes and create the directories for them if they don't already exist
	for _, vol := range version.Volumes {
		//Make sure that a path is given. If not we already assume that the working directory will be mounted
		if vol.Path != "" {
			err = ic.tools.MakeDir(ed + vol.Path)

			if err != nil {
				return err
			}
		}
	}

	//Check and see if any files need to be copied from the container to one of the volumes on the host.
	if len(version.Copies) > 0 {

		fmt.Println("Copying necessary files 1/3")
		//Create the container so that we can copy the files over to the right places
		containerID, err := ic.tools.CreateContainer(version.Image, cli)

		if err != nil {
			return err
		}

		fmt.Println("Copying necessary files 2/3")
		//Copy the files from the container to the locations
		for _, copy := range version.Copies {
			err = ic.tools.CopyFromContainer(copy.Source, ed+copy.Dest, containerID, cli, ic.cp)

			if err != nil {
				return err
			}
		}

		fmt.Println("Copying necessary files 3/3")
		//Remove the Container
		err = ic.tools.RemoveContainer(containerID, cli)

		if err != nil {
			return err
		}

	}

	if ic.config.Alias {
		//Set the alias for the command
		fmt.Println("Setting Alias")

		if runtime.GOOS == "windows" {
			if version.Version != "latest" {
				err = ic.tools.AddAliasWin(pim.Name+":"+version.Version, ed)
			} else {
				err = ic.tools.AddAliasWin(pim.Name, ed)
			}
		} else {
			if version.Version != "latest" {
				err = ic.tools.AddAliasUnix(pim.Name+":"+version.Version, ed)
			} else {
				err = ic.tools.AddAliasUnix(pim.Name, ed)
			}
		}

		if err != nil {
			return err
		}
	}

	fmt.Println(pim.Name, "successfully installed")

	return nil
}