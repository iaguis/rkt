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

// image package implements finding images in the store and fetching
// them from local or remote locations. The only API exposed are
// Finder and Fetcher - all their fields are also exposed (see action
// in common.go).
//
// Hacking docs:
//
// Documentation of a specific component is in a related file. Here,
// only the relations between components are described.
//
// At the lowest level there is a downloader with its downloadSession
// interface.
//
// On top of the above are two implementations of downloadSession
// interface - defaultDownloadSession (alongside the downloader) and
// resumableSession.
//
// Next to the downloadSession implementations there is asc,
// ascFetcher interface and two implementations of ascFetcher -
// localAscFetcher and remoteAscFetcher.
//
// On top of the above is httpOps. It provides a remoteAscFetcher
// instance.
//
// Next to httpOps there is validator.
//
// On top of the above there are namefetcher and httpfetcher. They use
// both httpOps and validator.
//
// Next to the above is filefetcher - it only uses validator.
//
// Next to the above is dockerfetcher.
//
// On top of the above is Fetcher.
//
// On top of the above is Finder.
package image
