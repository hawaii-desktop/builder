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

package webserver

import (
	"fmt"
	"github.com/plimble/ace"
	"github.com/plimble/utils/pool"
	"html/template"
	"net/http"
	"path/filepath"
)

// Template renderer.
type TemplateView struct {
	Templates  map[string]*template.Template
	bufferPool *pool.BufferPool
}

// Create template renderer.
func TemplateRenderer(path string) ace.Renderer {
	v := &TemplateView{
		Templates:  make(map[string]*template.Template),
		bufferPool: pool.NewBufferPool(64),
	}

	bases, err := filepath.Glob(path + "/*.tmpl")
	if err != nil {
		panic(err)
	}

	layouts, err := filepath.Glob(path + "/*.html")
	if err != nil {
		panic(err)
	}

	for _, layout := range layouts {
		files := append(bases, layout)
		v.Templates[filepath.Base(layout)] = template.Must(template.ParseFiles(files...))
	}

	return v
}

// Call the render function and panic on errors.
// The panic handler will catch the error and display a 500 error page.
func (v *TemplateView) Render(w http.ResponseWriter, name string, data interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}

	panic(v.render(w, name, data.(map[string]interface{})))
}

// Actually render the template view and return any error.
func (v *TemplateView) render(w http.ResponseWriter, name string, data map[string]interface{}) error {
	tmpl, ok := v.Templates[name]
	if !ok {
		return fmt.Errorf("template \"%s\" doesn't exist", name)
	}

	buffer := v.bufferPool.Get()
	defer v.bufferPool.Put(buffer)

	err := tmpl.ExecuteTemplate(buffer, "base", data)
	if err != nil {
		return err
	}

	buffer.WriteTo(w)
	return nil
}
