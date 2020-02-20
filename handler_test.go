// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"gotest.tools/v3/assert"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name   string
		config string
		path   string

		goImport string
		goSource string
		redirect string
	}{
		{
			name: "explicit display",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /portmidi:\n" +
				"    repo: https://github.com/rakyll/portmidi\n" +
				"    display: https://github.com/rakyll/portmidi _ _\n",
			path:     "/portmidi",
			goImport: "example.com/portmidi git https://github.com/rakyll/portmidi",
			goSource: "example.com/portmidi https://github.com/rakyll/portmidi _ _",
		},
		{
			name: "display GitHub inference",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /portmidi:\n" +
				"    repo: https://github.com/rakyll/portmidi\n",
			path:     "/portmidi",
			goImport: "example.com/portmidi git https://github.com/rakyll/portmidi",
			goSource: "example.com/portmidi https://github.com/rakyll/portmidi https://github.com/rakyll/portmidi/tree/master{/dir} https://github.com/rakyll/portmidi/blob/master{/dir}/{file}#L{line}",
		},
		{
			name: "Bitbucket Mercurial",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /gopdf:\n" +
				"    repo: https://bitbucket.org/zombiezen/gopdf\n" +
				"    vcs: hg\n",
			path:     "/gopdf",
			goImport: "example.com/gopdf hg https://bitbucket.org/zombiezen/gopdf",
			goSource: "example.com/gopdf https://bitbucket.org/zombiezen/gopdf https://bitbucket.org/zombiezen/gopdf/src/default{/dir} https://bitbucket.org/zombiezen/gopdf/src/default{/dir}/{file}#{file}-{line}",
		},
		{
			name: "Bitbucket Git",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /mygit:\n" +
				"    repo: https://bitbucket.org/zombiezen/mygit\n" +
				"    vcs: git\n",
			path:     "/mygit",
			goImport: "example.com/mygit git https://bitbucket.org/zombiezen/mygit",
			goSource: "example.com/mygit https://bitbucket.org/zombiezen/mygit https://bitbucket.org/zombiezen/mygit/src/default{/dir} https://bitbucket.org/zombiezen/mygit/src/default{/dir}/{file}#{file}-{line}",
		},
		{
			name: "subpath",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /portmidi:\n" +
				"    repo: https://github.com/rakyll/portmidi\n" +
				"    display: https://github.com/rakyll/portmidi _ _\n",
			path:     "/portmidi/foo",
			goImport: "example.com/portmidi git https://github.com/rakyll/portmidi",
			goSource: "example.com/portmidi https://github.com/rakyll/portmidi _ _",
		},
		{
			name: "subpath with trailing config slash",
			config: "host: example.com\n" +
				"paths:\n" +
				"  /portmidi/:\n" +
				"    repo: https://github.com/rakyll/portmidi\n" +
				"    display: https://github.com/rakyll/portmidi _ _\n",
			path:     "/portmidi/foo",
			goImport: "example.com/portmidi git https://github.com/rakyll/portmidi",
			goSource: "example.com/portmidi https://github.com/rakyll/portmidi _ _",
		},
		{
			name:     "gotest.tools",
			config:   gotestToolsConfig,
			path:     "/",
			goImport: "gotest.tools git https://github.com/gotestyourself/gotest.tools",
			goSource: "gotest.tools https://github.com/gotestyourself/gotest.tools https://github.com/gotestyourself/gotest.tools/tree/master{/dir} https://github.com/gotestyourself/gotest.tools/blob/master{/dir}/{file}#L{line}",
			redirect: "https://pkg.go.dev/gotest.tools/v3/",
		},
		{
			name:     "gotest.tools/gotestsum",
			config:   gotestToolsConfig,
			path:     "/gotestsum",
			goImport: "gotest.tools/gotestsum git https://github.com/gotestyourself/gotestsum",
			goSource: "gotest.tools/gotestsum https://github.com/gotestyourself/gotestsum https://github.com/gotestyourself/gotestsum/tree/master{/dir} https://github.com/gotestyourself/gotestsum/blob/master{/dir}/{file}#L{line}",
			redirect: "https://pkg.go.dev/gotest.tools/gotestsum/",
		},
		{
			name:     "gotest.tools/assert",
			config:   gotestToolsConfig,
			path:     "/assert",
			goImport: "gotest.tools git https://github.com/gotestyourself/gotest.tools",
			goSource: "gotest.tools https://github.com/gotestyourself/gotest.tools https://github.com/gotestyourself/gotest.tools/tree/master{/dir} https://github.com/gotestyourself/gotest.tools/blob/master{/dir}/{file}#L{line}",
			redirect: "https://pkg.go.dev/gotest.tools/v3/assert",
		},
		{
			name:     "gotest.tools/v5/assert",
			config:   gotestToolsConfig,
			path:     "/v5/assert",
			goImport: "gotest.tools git https://github.com/gotestyourself/gotest.tools",
			goSource: "gotest.tools https://github.com/gotestyourself/gotest.tools https://github.com/gotestyourself/gotest.tools/tree/master{/dir} https://github.com/gotestyourself/gotest.tools/blob/master{/dir}/{file}#L{line}",
			redirect: "https://pkg.go.dev/gotest.tools/v5/assert",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, err := newHandler([]byte(tc.config))
			assert.NilError(t, err)

			s := httptest.NewServer(h)
			defer s.Close()

			resp, err := http.Get(s.URL + tc.path)
			assert.NilError(t, err)

			data, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			assert.NilError(t, err)
			assert.Equal(t, resp.StatusCode, http.StatusOK)

			goImport := findMeta(data, "go-import")
			assert.Equal(t, goImport, tc.goImport, "go import")

			goSource := findMeta(data, "go-source")
			assert.Equal(t, goSource, tc.goSource, "go source")

			if tc.redirect != "" {
				redirect := findRedirect(data)
				assert.Equal(t, redirect, tc.redirect, "redirect")
			}
		})
	}
}

var gotestToolsConfig = `
host: gotest.tools
paths:
  /:
    repo: https://github.com/gotestyourself/gotest.tools
    default_version: v3
  /gotestsum:
    repo: https://github.com/gotestyourself/gotestsum
`

func TestBadConfigs(t *testing.T) {
	badConfigs := []string{
		"paths:\n" +
			"  /missingvcs:\n" +
			"    repo: https://bitbucket.org/zombiezen/gopdf\n",
		"paths:\n" +
			"  /unknownvcs:\n" +
			"    repo: https://bitbucket.org/zombiezen/gopdf\n" +
			"    vcs: xyzzy\n",
		"cache_max_age: -1\n" +
			"paths:\n" +
			"  /portmidi:\n" +
			"    repo: https://github.com/rakyll/portmidi\n",
	}
	for _, config := range badConfigs {
		_, err := newHandler([]byte(config))
		if err == nil {
			t.Errorf("expected config to produce an error, but did not:\n%s", config)
		}
	}
}

func findMeta(data []byte, name string) string {
	var sep []byte
	sep = append(sep, `<meta name="`...)
	sep = append(sep, name...)
	sep = append(sep, `" content="`...)
	i := bytes.Index(data, sep)
	if i == -1 {
		return ""
	}
	content := data[i+len(sep):]
	j := bytes.IndexByte(content, '"')
	if j == -1 {
		return ""
	}
	return string(content[:j])
}

func findRedirect(data []byte) string {
	sep := []byte(`<meta http-equiv="refresh" content="0; url=`)
	i := bytes.Index(data, sep)
	if i == -1 {
		return ""
	}
	content := data[i+len(sep):]
	j := bytes.IndexByte(content, '"')
	if j == -1 {
		return ""
	}
	return string(content[:j])
}

func TestPathConfigSetFind(t *testing.T) {
	tests := []struct {
		paths   []string
		query   string
		want    string
		subpath string
	}{
		{
			paths: []string{"/portmidi"},
			query: "/portmidi",
			want:  "/portmidi",
		},
		{
			paths: []string{"/portmidi"},
			query: "/portmidi/",
			want:  "/portmidi",
		},
		{
			paths: []string{"/portmidi"},
			query: "/foo",
			want:  "",
		},
		{
			paths: []string{"/portmidi"},
			query: "/zzz",
			want:  "",
		},
		{
			paths: []string{"/abc", "/portmidi", "/xyz"},
			query: "/portmidi",
			want:  "/portmidi",
		},
		{
			paths:   []string{"/abc", "/portmidi", "/xyz"},
			query:   "/portmidi/foo",
			want:    "/portmidi",
			subpath: "foo",
		},
		{
			paths:   []string{"/example/helloworld", "/", "/y", "/foo"},
			query:   "/x",
			want:    "/",
			subpath: "x",
		},
		{
			paths:   []string{"/example/helloworld", "/", "/y", "/foo"},
			query:   "/",
			want:    "/",
			subpath: "",
		},
		{
			paths:   []string{"/example/helloworld", "/", "/y", "/foo"},
			query:   "/example",
			want:    "/",
			subpath: "example",
		},
		{
			paths:   []string{"/example/helloworld", "/", "/y", "/foo"},
			query:   "/example/foo",
			want:    "/",
			subpath: "example/foo",
		},
		{
			paths: []string{"/example/helloworld", "/", "/y", "/foo"},
			query: "/y",
			want:  "/y",
		},
		{
			paths:   []string{"/example/helloworld", "/", "/y", "/foo"},
			query:   "/x/y/",
			want:    "/",
			subpath: "x/y/",
		},
		{
			paths: []string{"/example/helloworld", "/y", "/foo"},
			query: "/x",
			want:  "",
		},
	}
	emptyToNil := func(s string) string {
		if s == "" {
			return "<nil>"
		}
		return s
	}
	for _, test := range tests {
		pset := make(pathConfigSet, len(test.paths))
		for i := range test.paths {
			pset[i].path = test.paths[i]
		}
		sort.Sort(pset)
		pc, subpath := pset.find(test.query)
		var got string
		if pc != nil {
			got = pc.path
		}
		if got != test.want || subpath != test.subpath {
			t.Errorf("pathConfigSet(%v).find(%q) = %v, %v; want %v, %v",
				test.paths, test.query, emptyToNil(got), subpath, emptyToNil(test.want), test.subpath)
		}
	}
}

func TestCacheHeader(t *testing.T) {
	tests := []struct {
		name         string
		config       string
		cacheControl string
	}{
		{
			name:         "default",
			cacheControl: "public, max-age=86400",
		},
		{
			name:         "specify time",
			config:       "cache_max_age: 60\n",
			cacheControl: "public, max-age=60",
		},
		{
			name:         "zero",
			config:       "cache_max_age: 0\n",
			cacheControl: "public, max-age=0",
		},
	}
	for _, test := range tests {
		h, err := newHandler([]byte("paths:\n  /portmidi:\n    repo: https://github.com/rakyll/portmidi\n" +
			test.config))
		if err != nil {
			t.Errorf("%s: newHandler: %v", test.name, err)
			continue
		}
		s := httptest.NewServer(h)
		resp, err := http.Get(s.URL + "/portmidi")
		if err != nil {
			t.Errorf("%s: http.Get: %v", test.name, err)
			continue
		}
		resp.Body.Close()
		got := resp.Header.Get("Cache-Control")
		if got != test.cacheControl {
			t.Errorf("%s: Cache-Control header = %q; want %q", test.name, got, test.cacheControl)
		}
	}
}
