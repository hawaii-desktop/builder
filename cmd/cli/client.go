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
	"errors"
	"fmt"
	pb "github.com/hawaii-desktop/builder/protocol"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
	"regexp"
	"strings"
)

// Store client stuff.
type Client struct {
	// Connection.
	conn *grpc.ClientConn
	// RPC proxy.
	client pb.BuilderClient
}

var (
	ErrFailed     = errors.New("master didn't satisfy our request")
	ErrInvalidVcs = errors.New("invalid VCS information")
)

const (
	AllChroots      pb.EnumListChroots = pb.EnumListChroots_AllChroots
	ActiveChroots   pb.EnumListChroots = pb.EnumListChroots_ActiveChroots
	InactiveChroots pb.EnumListChroots = pb.EnumListChroots_InactiveChroots
)

// Create a new Client object.
func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{conn: conn, client: pb.NewBuilderClient(conn)}
}

// Add a chroot.
func (c *Client) AddChroot(release, version, arch string) error {
	// Send message
	args := &pb.ChrootInfo{release, version, arch}
	reply, err := c.client.AddChroot(context.Background(), args)
	if err != nil {
		return err
	}
	if !reply.Result {
		return ErrFailed
	}
	return nil
}

// Remove chroot.
func (c *Client) RemoveChroot(release, version, arch string) error {
	args := &pb.ChrootInfo{release, version, arch}
	reply, err := c.client.RemoveChroot(context.Background(), args)
	if err != nil {
		return err
	}
	if !reply.Result {
		return ErrFailed
	}
	return nil
}

// List chroots.
func (c *Client) ListChroots(state_flag pb.EnumListChroots) error {
	stream, err := c.client.ListChroots(context.Background(), &pb.ListChrootsRequest{state_flag})
	if err != nil {
		return err
	}

	for {
		chroot, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fmt.Printf("Chroot \"%s-%s-%s\"\n", chroot.Release, chroot.Version, chroot.Architecture)
	}

	return nil
}

// Add a package.
func (c *Client) AddPackage(name string, archs string, ci bool, vcs string, uvcs string) error {
	// Split architectures
	a := strings.Split(archs, ",")

	// Clean VCS strings
	if m, _ := regexp.MatchString("#branch=.+$", vcs); !m {
		vcs += "#branch=master"
	}
	if ci {
		if m, _ := regexp.MatchString("#branch=.+$", uvcs); !m {
			uvcs += "#branch=master"
		}
	}

	// VCS regexp
	r := regexp.MustCompile("(.+)#branch=(.+)$")

	// Decode VCS
	matches := r.FindStringSubmatch(vcs)
	if len(matches) != 3 {
		return ErrInvalidVcs
	}
	vcs_url := matches[1]
	vcs_branch := matches[2]

	// Decode upstream VCS
	var uvcs_url, uvcs_branch string
	if ci {
		matches = r.FindStringSubmatch(uvcs)
		if len(matches) != 3 {
			return ErrInvalidVcs
		}
		uvcs_url = matches[1]
		uvcs_branch = matches[2]
	}

	// Send message
	args := &pb.PackageInfo{name, a, ci, &pb.VcsInfo{vcs_url, vcs_branch}, &pb.VcsInfo{uvcs_url, uvcs_branch}}
	reply, err := c.client.AddPackage(context.Background(), args)
	if err != nil {
		return err
	}
	if !reply.Result {
		return ErrFailed
	}
	return nil
}

// Remove package.
func (c *Client) RemovePackage(name string) error {
	args := &pb.StringMessage{name}
	reply, err := c.client.RemovePackage(context.Background(), args)
	if err != nil {
		return err
	}
	if !reply.Result {
		return ErrFailed
	}
	return nil
}

// List packages.
func (c *Client) ListPackages() error {
	stream, err := c.client.ListPackages(context.Background(), &pb.StringMessage{".+"})
	if err != nil {
		return err
	}

	for {
		pkg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fmt.Printf("Package \"%s\"\n", pkg.Name)
		fmt.Printf("\tArchitectures: %s\n", strings.Join(pkg.Architectures, ", "))
		fmt.Printf("\tCI: %v\n", pkg.Ci)
		fmt.Println("\tVCS:")
		fmt.Printf("\t\tURL: %s\n", pkg.Vcs.Url)
		fmt.Printf("\t\tBranch: %s\n", pkg.Vcs.Branch)
		if pkg.Ci {
			fmt.Println("\tUpstream VCS:")
			fmt.Printf("\t\tURL: %s\n", pkg.UpstreamVcs.Url)
			fmt.Printf("\t\tBranch: %s\n", pkg.UpstreamVcs.Branch)
		}
	}

	return nil
}

// Add an image.
func (c *Client) AddImage(name, descr, archs, vcs string) error {
	// Split architectures
	a := strings.Split(archs, ",")

	// Clean VCS strings
	if m, _ := regexp.MatchString("#branch=.+$", vcs); !m {
		vcs += "#branch=master"
	}

	// VCS regexp
	r := regexp.MustCompile("(.+)#branch=(.+)$")

	// Decode VCS
	matches := r.FindStringSubmatch(vcs)
	if len(matches) != 3 {
		return ErrInvalidVcs
	}
	vcs_url := matches[1]
	vcs_branch := matches[2]

	// Send message
	args := &pb.ImageInfo{name, descr, a, &pb.VcsInfo{vcs_url, vcs_branch}}
	reply, err := c.client.AddImage(context.Background(), args)
	if err != nil {
		return err
	}
	if !reply.Result {
		return ErrFailed
	}
	return nil
}

// Remove image.
func (c *Client) RemoveImage(name string) error {
	args := &pb.StringMessage{name}
	reply, err := c.client.RemoveImage(context.Background(), args)
	if err != nil {
		return err
	}
	if !reply.Result {
		return ErrFailed
	}
	return nil
}

// List images.
func (c *Client) ListImages() error {
	stream, err := c.client.ListImages(context.Background(), &pb.StringMessage{".+"})
	if err != nil {
		return err
	}

	for {
		img, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fmt.Printf("Image \"%s\"\n", img.Name)
		fmt.Printf("\tDescription: %v\n", img.Description)
		fmt.Printf("\tArchitectures: %s\n", strings.Join(img.Architectures, ", "))
		fmt.Println("\tVCS:")
		fmt.Printf("\t\tURL: %s\n", img.Vcs.Url)
		fmt.Printf("\t\tBranch: %s\n", img.Vcs.Branch)
	}

	return nil
}

// Schedule a job.
func (c *Client) SendJob(target, arch, tstr string) (uint64, error) {
	var t pb.EnumTargetType
	switch tstr {
	case "package":
		t = pb.EnumTargetType_PACKAGE
		break
	case "image":
		t = pb.EnumTargetType_IMAGE
		break
	default:
		return 0, ErrWrongArguments
	}

	args := &pb.CollectJobRequest{Target: target, Architecture: arch, Type: t}
	reply, err := c.client.CollectJob(context.Background(), args)
	if err != nil {
		return 0, err
	}
	if !reply.Result {
		return 0, ErrFailed
	}
	return reply.Id, nil
}

// Close client connection.
func (c *Client) Close() {
	c.conn.Close()
	c.conn = nil
	c.client = nil
}
