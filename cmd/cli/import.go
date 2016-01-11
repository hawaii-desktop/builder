/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015-2016 Pier Luigi Fiorini
 *
 * Author(s):
 *    Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * $BEGIN_LICENSE:AGPL3+$
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * $END_LICENSE$
 ***************************************************************************/

package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/hawaii-desktop/builder/logging"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type VcsInfo struct {
	Url    string `yaml:"url"`
	Branch string `yaml:"branch"`
}

type PackageEntry struct {
	Name          string   `yaml:"name"`
	Architectures []string `yaml:"archs"`
	Ci            bool     `yaml:"ci"`
	Vcs           VcsInfo  `yaml:"vcs"`
	UpstreamVcs   VcsInfo  `yaml:"uvcs"`
	Disabled      bool     `yaml:"disabled"`
}

type ImageEntry struct {
	Name          string   `yaml:"name"`
	Description   string   `yaml:"descr"`
	Architectures []string `yaml:"archs"`
	Vcs           VcsInfo  `yaml:"vcs"`
	Disabled      bool     `yaml:"disabled"`
}

type Data struct {
	AddPackages    []PackageEntry `yaml:"add-packages"`
	RemovePackages []string       `yaml:"remove-packages"`
	AddImages      []ImageEntry   `yaml:"add-images"`
	RemoveImages   []string       `yaml:"remove-images"`
}

var CmdImport = cli.Command{
	Name:        "import",
	Usage:       "Add and remove packages and images from file",
	Description: `Add and remove packages and image from a YAML file.`,
	Before: func(ctx *cli.Context) error {
		if !ctx.IsSet("filename") {
			logging.Errorln("You must specify the file to import")
			return ErrWrongArguments
		}
		return nil
	},
	Action: runImport,
	Flags: []cli.Flag{
		cli.StringFlag{"filename, f", "", "file to import", ""},
	},
}

const defaultArchitectures = "i386,x86_64,armhfp"

func runImport(ctx *cli.Context) {
	// Open file
	yamlFile, err := ioutil.ReadFile(ctx.String("filename"))
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Unmarshal
	var data Data
	err = yaml.Unmarshal(yamlFile, &data)
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Connect to the master
	conn, err := grpc.Dial(Config.Master.Address, grpc.WithInsecure())
	if err != nil {
		logging.Errorln(err)
		return
	}

	// Create client proxy
	client := NewClient(conn)
	defer client.Close()

	// Process all the packages to add
	for _, pkg := range data.AddPackages {
		if pkg.Disabled {
			continue
		}

		archs := defaultArchitectures
		if len(pkg.Architectures) > 0 {
			archs = strings.Join(pkg.Architectures, ",")
		}

		if pkg.Vcs.Branch == "" {
			pkg.Vcs.Branch = "master"
		}
		vcs := fmt.Sprintf("%s#branch=%s", pkg.Vcs.Url, pkg.Vcs.Branch)

		uvcs := ""
		if pkg.Ci {
			if pkg.UpstreamVcs.Branch == "" {
				pkg.UpstreamVcs.Branch = "master"
			}
			uvcs = fmt.Sprintf("%s#branch=%s", pkg.UpstreamVcs.Url, pkg.UpstreamVcs.Branch)
		}

		if err = client.AddPackage(pkg.Name, archs, pkg.Ci, vcs, uvcs); err != nil {
			logging.Errorf("Failed to add package \"%s\": %s\n", pkg.Name, err)
		}
	}

	// Process all the packages to remove
	for _, name := range data.RemovePackages {
		if err = client.RemovePackage(name); err != nil {
			logging.Errorf("Failed to remove package \"%s\": %s\n", name, err)
		}
	}

	// Process all the images to add
	for _, img := range data.AddImages {
		if img.Disabled {
			continue
		}

		archs := defaultArchitectures
		if len(img.Architectures) > 0 {
			archs = strings.Join(img.Architectures, ",")
		}

		if img.Vcs.Branch == "" {
			img.Vcs.Branch = "master"
		}
		vcs := fmt.Sprintf("%s#branch=%s", img.Vcs.Url, img.Vcs.Branch)

		if err = client.AddImage(img.Name, img.Description, archs, vcs); err != nil {
			logging.Errorf("Failed to add image \"%s\": %s\n", img.Name, err)
		}
	}

	// Process all the images to remove
	for _, name := range data.RemoveImages {
		if err = client.RemoveImage(name); err != nil {
			logging.Errorf("Failed to remove image \"%s\": %s\n", name, err)
		}
	}
}
