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

package cli

import (
	"errors"
	pb "github.com/hawaii-desktop/builder/common/protocol"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Store client stuff.
type Client struct {
	// RPC proxy.
	client pb.BuilderClient
}

var (
	ErrFailed = errors.New("master failed to enqueue job")
)

// Create a new Client object.
func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{pb.NewBuilderClient(conn)}
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
