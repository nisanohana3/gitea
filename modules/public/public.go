// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package public

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"code.gitea.io/gitea/modules/container"
	"code.gitea.io/gitea/modules/httpcache"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
)

// Options represents the available options to configure the handler.
type Options struct {
	Directory   string
	Prefix      string
	CorsHandler func(http.Handler) http.Handler
}

// AssetsURLPathPrefix is the path prefix for static asset files
const AssetsURLPathPrefix = "/assets/"

// AssetsHandlerFunc implements the static handler for serving custom or original assets.
func AssetsHandlerFunc(opts *Options) http.HandlerFunc {
	custPath := filepath.Join(setting.CustomPath, "public")
	if !filepath.IsAbs(custPath) {
		custPath = filepath.Join(setting.AppWorkPath, custPath)
	}
	if !filepath.IsAbs(opts.Directory) {
		opts.Directory = filepath.Join(setting.AppWorkPath, opts.Directory)
	}
	if !strings.HasSuffix(opts.Prefix, "/") {
		opts.Prefix += "/"
	}

	return func(resp http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" && req.Method != "HEAD" {
			resp.WriteHeader(http.StatusNotFound)
			return
		}

		if opts.CorsHandler != nil {
			var corsSent bool
			opts.CorsHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				corsSent = true
			})).ServeHTTP(resp, req)
			// If CORS is not sent, the response must have been written by other handlers
			if !corsSent {
				return
			}
		}

		file := req.URL.Path[len(opts.Prefix):]

		// custom files
		if opts.handle(resp, req, http.Dir(custPath), file) {
			return
		}

		// internal files
		if opts.handle(resp, req, fileSystem(opts.Directory), file) {
			return
		}

		resp.WriteHeader(http.StatusNotFound)
	}
}

// parseAcceptEncoding parse Accept-Encoding: deflate, gzip;q=1.0, *;q=0.5 as compress methods
func parseAcceptEncoding(val string) container.Set[string] {
	parts := strings.Split(val, ";")
	types := make(container.Set[string])
	for _, v := range strings.Split(parts[0], ",") {
		types.Add(strings.TrimSpace(v))
	}
	return types
}

// setWellKnownContentType will set the Content-Type if the file is a well-known type.
// See the comments of detectWellKnownMimeType
func setWellKnownContentType(w http.ResponseWriter, file string) {
	mimeType := detectWellKnownMimeType(filepath.Ext(file))
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}
}

func (opts *Options) handle(w http.ResponseWriter, req *http.Request, fs http.FileSystem, file string) bool {
	// actually, fs (http.FileSystem) is designed to be a safe interface, relative paths won't bypass its parent directory, it's also fine to do a clean here
	f, err := fs.Open(util.PathJoinRelX(file))
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		w.WriteHeader(http.StatusInternalServerError)
		log.Error("[Static] Open %q failed: %v", file, err)
		return true
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error("[Static] %q exists, but fails to open: %v", file, err)
		return true
	}

	// Try to serve index file
	if fi.IsDir() {
		w.WriteHeader(http.StatusNotFound)
		return true
	}

	if httpcache.HandleFileETagCache(req, w, fi) {
		return true
	}

	setWellKnownContentType(w, file)

	serveContent(w, req, fi, fi.ModTime(), f)
	return true
}
