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

package webserver

import (
	"github.com/plimble/ace"
	"html/template"
	"net/http"
)

// Template renderer.
type TemplateView struct{}

func TemplateRenderer() ace.Renderer {
	return &TemplateView{}
}

// Render the template view.
func (v *TemplateView) Render(w http.ResponseWriter, name string, data interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}

	tmpl := template.Must(template.ParseFiles(name))
	tmpl.Execute(w, data.(map[string]interface{}))
}
