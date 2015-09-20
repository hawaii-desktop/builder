/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015 Pier Luigi Fiorini
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
	"github.com/hawaii-desktop/builder/src/logging"
	pb "github.com/hawaii-desktop/builder/src/protocol"
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

// Create a new Client object.
func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{conn: conn, client: pb.NewBuilderClient(conn)}
}

// Add a package.
func (c *Client) AddPackage(name string, archs string, ci bool, vcs string, uvcs string) error {
	// Split architectures
	a := strings.Split(archs, ",")

	// VCS regexp
	r := regexp.MustCompile("(.+)(#branch=.+)*$")

	// Decode VCS
	var vcs_url, vcs_branch string
	matches := r.FindStringSubmatch(vcs)
	if len(matches) == 1 {
		return ErrInvalidVcs
	}
	vcs_url = matches[1]
	if len(matches) > 2 {
		vcs_branch = strings.Replace(matches[2], "#branch=", "", 1)
	}
	if vcs_branch == "" {
		vcs_branch = "master"
	}

	// Decode upstream VCS
	var uvcs_url, uvcs_branch string
	if ci {
		matches = r.FindStringSubmatch(uvcs)
		if len(matches) == 1 {
			return ErrInvalidVcs
		}
		uvcs_url = matches[1]
		if len(matches) > 2 {
			uvcs_branch = strings.Replace(matches[2], "#branch=", "", 1)
		}
		if uvcs_branch == "" {
			uvcs_branch = "master"
		}
	}

	// Send message
	logging.Infof("Adding package \"%s\" (architectures: %q, ci: %v, vcs url: %v, vcs branch: %v, upstream vcs url: %v, upstream vcs branch: %v)",
		name, a, ci, vcs_url, vcs_branch, uvcs_url, uvcs_branch)
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

// Schedule a job.
func (c *Client) SendJob(target string) (uint64, error) {
	args := &pb.CollectJobRequest{Target: target}
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
