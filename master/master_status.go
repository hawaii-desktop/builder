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

package master

import (
	"fmt"
	"github.com/hawaii-desktop/builder"
	"github.com/hawaii-desktop/builder/status"
)

// Send status notifications to the configured recipients.
func (m *Master) sendStatusNotifications(job *Job) {
	if Config.Notifications.Slack {
		slack := status.Slack(Config.Slack.URL)
		color := "good"
		if job.Status >= builder.JOB_STATUS_FAILED {
			color = "danger"
		}
		text := fmt.Sprintf("Job <http://build.hawaiios.org/job/%d|#%d> has finished", job.Id, job.Id)
		slack.Send(&status.SlackMessage{
			Attachments: []*status.SlackAttachment{
				&status.SlackAttachment{
					Fallback: fmt.Sprintf("Status: %s", builder.JobStatusDescriptionMap[job.Status]),
					Pretext:  text,
					Color:    color,
					Fields: []*status.SlackFields{
						&status.SlackFields{
							Title: "Target",
							Value: fmt.Sprintf("%s (%s)", job.Target, job.Architecture),
							Short: false,
						},
						&status.SlackFields{
							Title: "Elapsed Time",
							Value: fmt.Sprintf("%s", job.Finished.Sub(job.Started)),
							Short: false,
						},
						&status.SlackFields{
							Title: "Status",
							Value: builder.JobStatusDescriptionMap[job.Status],
							Short: false,
						},
					},
				},
			},
		})
	}
}
