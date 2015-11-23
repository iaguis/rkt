// Copyright 2015 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package image

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/coreos/rkt/rkt/config"
	"github.com/coreos/rkt/store"
)

// httpOps is a kind of facade around downloader and a
// resumableSession. It provides some higher-level functions for
// fetching images and signature keys. It also is a provider of a
// remote fetcher for asc.
type httpOps struct {
	InsecureSkipTLSVerify bool
	S                     *store.Store
	Headers               map[string]config.Headerer
}

// DownloadSignature takes an asc instance and tries to get the
// signature. If the remote server asked to to defer the download,
// this function will return true and no error and no file.
func (o *httpOps) DownloadSignature(a *asc) (readSeekCloser, bool, error) {
	stderr("downloading signature from %v", a.Location)
	ascFile, err := a.Get()
	if err == nil {
		return ascFile, false, nil
	}
	if _, ok := err.(*statusAcceptedError); ok {
		stderr("server requested deferring the signature download")
		return nil, true, nil
	}
	return nil, false, fmt.Errorf("error downloading the signature file: %v", err)
}

// DownloadSignatureAgain again does similar thing to
// DownloadSignature, but it expects the signature to be actually
// provided, that is - no deferring this time.
func (o *httpOps) DownloadSignatureAgain(a *asc) (readSeekCloser, error) {
	ascFile, retry, err := o.DownloadSignature(a)
	if err != nil {
		return nil, err
	}
	if retry {
		return nil, fmt.Errorf("error downloading the signature file: server asked to defer the download again")
	}
	return ascFile, nil
}

// DownloadImage download the image, duh. It expects to actually
// receive the file, instead of being asked to use the cached version.
func (o *httpOps) DownloadImage(u *url.URL) (readSeekCloser, *cacheData, error) {
	image, cd, err := o.DownloadImageWithETag(u, "")
	if err != nil {
		return nil, nil, err
	}
	if cd.UseCached {
		return nil, nil, fmt.Errorf("asked to use cached image even if not asked for that")
	}
	return image, cd, nil
}

// DownloadImageWithETag might download an image or tell you to use
// the cached image. In the latter case the returned file will be nil.
func (o *httpOps) DownloadImageWithETag(u *url.URL, etag string) (readSeekCloser, *cacheData, error) {
	aciFile, err := getTmpROC(o.S, u.String())
	if err != nil {
		return nil, nil, err
	}
	defer func() { maybeClose(aciFile) }()

	session := o.getSession(u, aciFile.File, "ACI", etag)
	dl := o.getDownloader(session)
	if err := dl.Download(u, aciFile.File); err != nil {
		return nil, nil, fmt.Errorf("error downloading ACI: %v", err)
	}
	if session.Cd.UseCached {
		return nil, session.Cd, nil
	}
	retAciFile := aciFile
	aciFile = nil
	return retAciFile, session.Cd, nil
}

// GetAscRemoteFetcher provides a remoteAscFetcher for asc.
func (o *httpOps) GetAscRemoteFetcher() *remoteAscFetcher {
	f := func(u *url.URL, file *os.File) error {
		switch u.Scheme {
		case "http", "https":
		default:
			return fmt.Errorf("invalid signature location: expected %q scheme, got %q", "http(s)", u.Scheme)
		}
		session := o.getSession(u, file, "signature", "")
		dl := o.getDownloader(session)
		err := dl.Download(u, file)
		if err != nil {
			return err
		}
		if session.Cd.UseCached {
			return fmt.Errorf("unexpected cache reuse request for signature %q", u.String())
		}
		return nil
	}
	return &remoteAscFetcher{
		F: f,
		S: o.S,
	}
}

func (o *httpOps) getSession(u *url.URL, file *os.File, label, etag string) *resumableSession {
	eTagFilePath := fmt.Sprintf("%s.etag", file.Name())
	return &resumableSession{
		InsecureSkipTLSVerify: o.InsecureSkipTLSVerify,
		Headers:               o.getHeaders(u, etag),
		File:                  file,
		ETagFilePath:          eTagFilePath,
		Label:                 label,
	}
}

func (o *httpOps) getDownloader(session downloadSession) *downloader {
	return &downloader{
		Session: session,
	}
}

func (o *httpOps) getHeaders(u *url.URL, etag string) http.Header {
	options := o.getHeadersForURL(u)
	if etag != "" {
		options.Add("If-None-Match", etag)
	}
	return options
}

func (o *httpOps) getHeadersForURL(u *url.URL) http.Header {
	// Send credentials only over secure channel
	// TODO(krnowak): This could be controlled with another
	// insecure flag.
	if u.Scheme == "https" {
		if hostOpts, ok := o.Headers[u.Host]; ok {
			return hostOpts.Header()
		}
	}

	return make(http.Header)
}
