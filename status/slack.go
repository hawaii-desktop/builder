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

package status

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Slack string

type SlackMessage struct {
	Channel     string             `json:"channel,omitempty"`
	UserName    string             `json:"username,omitempty"`
	IconEmoji   string             `json:"icon_emoji,omitempty"`
	Attachments []*SlackAttachment `json:"attachments,omitempty"`
	Text        string             `json:"text"`
}

type SlackAttachment struct {
	Fallback string         `json:"fallback"`
	Pretext  string         `json:"pretext,omitempty"`
	Color    string         `json:"color,omitempty"`
	Fields   []*SlackFields `json:"fields,omitempty"`
}

type SlackFields struct {
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
	Short bool   `json:"short"`
}

func (s Slack) String() string {
	return string(s)
}

func (s Slack) Send(msg *SlackMessage) error {
	// Marshal JSON
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Make a POST request
	req, err := http.NewRequest("POST", s.String(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	// Set appropriate content type
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if !bytes.Equal(body, []byte("ok")) {
		return fmt.Errorf("Slack replied with an error: %s", string(body))
	}

	return nil
}
