// Copyright (c) 2021 Shohruh Adham
// Use of this source code is governed by the MIT License.

package nanomux

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// --------------------------------------------------

// URLTmpl is used to keep the resource's scheme, host and prefix path segments.
type URLTmpl struct {
	Scheme     string
	Host       string
	PrefixPath string
}

var rootTmpl = Parse("/")

// --------------------------------------------------

type _Parent interface {
	parent() _Parent
}

// --------------------------------------------------

var slashDecoderRe = regexp.MustCompile("(%2f)|(%2F)")

type _PathSegmentIterator struct {
	path   string
	root   bool
	tslash bool
}

func makePathSegmentIterator(path string) _PathSegmentIterator {
	return _PathSegmentIterator{
		path:   path,
		root:   path == "/",
		tslash: path != "" && path != "/" && path[len(path)-1] == '/',
	}
}

func (psi *_PathSegmentIterator) pathIsRoot() bool {
	return psi.root
}

func (psi *_PathSegmentIterator) nextSegment() string {
	if psi.path == "" {
		return ""
	}

	var segment string
	if psi.path[0] == '/' {
		// This if statement runs only once.
		// The iterator doesn't keep the leading slash.
		psi.path = psi.path[1:]
	}

	var idx = strings.IndexByte(psi.path, '/')
	if idx < 0 {
		segment = psi.path
		psi.path = ""
	} else {
		segment = psi.path[:idx]
		psi.path = psi.path[idx+1:]
	}

	return slashDecoderRe.ReplaceAllLiteralString(segment, "/")
}

func (psi *_PathSegmentIterator) remainingPath() string {
	// Remaining path is returned without leading slash.
	if psi.path != "" && psi.path[0] == '/' {
		psi.path = psi.path[1:]
	}

	return psi.path
}

func (psi *_PathSegmentIterator) pathHasTrailingSlash() bool {
	return psi.tslash
}

// --------------------------------------------------

func cloneRequestURL(r *http.Request) *url.URL {
	var url = &url.URL{}
	*url = *r.URL
	return url
}

func toUpperSplitByCommaSpace(str string) []string {
	str = strings.TrimSpace(str)
	var strs []string
	for idx, splitStr := 0, ""; len(str) > 0; {
		idx = strings.IndexAny(str, ", ")
		if idx == 0 {
			str = str[1:]
			continue
		}

		if idx > 0 {
			splitStr = strings.ToUpper(str[:idx])
			str = str[idx+1:]
		} else {
			splitStr = strings.ToUpper(str)
			str = ""
		}

		strs = append(strs, splitStr)
	}

	return strs
}

// splitHostAndPath splits the URL template into the host and path templates.
// It also returns the security and tslash property values.
//
// Only an absolute URL template can have a host template. When a URL template
// doesn't start with a scheme, it is considered a path template. After a host
// template, if a path template contains only a slash, the trailing slash return
// value will be true and the path template return value will be empty.
//
// A URL template can start with a scheme to specify security even if there is
// no host template.
//
// For example,
// https:///resource1/resource2 - specifies that resource2 is secure.
// http:///resource1/ - specifies that resource1 is insecure and has a tslash.
func splitHostAndPath(urlTmplStr string) (
	hostTmplStr, pathTmplStr string,
	secure bool,
	tslash bool,
	err error,
) {
	if urlTmplStr == "" {
		err = ErrInvalidTemplate
		return
	}

	var absolute bool
	var lurlTmplStr = len(urlTmplStr)
	urlTmplStr = strings.TrimPrefix(urlTmplStr, "https://")
	if len(urlTmplStr) < lurlTmplStr {
		absolute = true
		secure = true
	} else {
		urlTmplStr = strings.TrimPrefix(urlTmplStr, "http://")
		if len(urlTmplStr) < lurlTmplStr {
			absolute = true
		}
	}

	if absolute {
		var idx = strings.IndexByte(urlTmplStr, '/')
		if idx < 0 {
			hostTmplStr = urlTmplStr
			return
		}

		hostTmplStr = urlTmplStr[:idx]
		pathTmplStr = urlTmplStr[idx:]
	} else {
		pathTmplStr = urlTmplStr
	}

	if hostTmplStr != "" {
		if pathTmplStr == "/" {
			tslash = true
			pathTmplStr = ""
			return
		}
	} else if pathTmplStr == "" {
		secure = false
		err = ErrInvalidTemplate
		return
	}

	var lastIdx = len(pathTmplStr) - 1
	if lastIdx > 0 {
		if pathTmplStr[lastIdx] == '/' {
			tslash = true
			pathTmplStr = pathTmplStr[:lastIdx]
		}
	}

	return
}

func splitURL(urlTmplStr string) (
	hostTmplStr string,
	prefixPathTmplStr string,
	resourceTmplStr string,
	secure bool,
	tslash bool,
	err error,
) {
	hostTmplStr, urlTmplStr, secure, tslash, err = splitHostAndPath(urlTmplStr)
	if err != nil {
		return
	}

	if urlTmplStr == "" || urlTmplStr == "/" {
		resourceTmplStr = urlTmplStr
		return
	}

	var idx = strings.LastIndexByte(urlTmplStr, '/')
	if idx < 0 {
		resourceTmplStr = slashDecoderRe.ReplaceAllLiteralString(
			urlTmplStr,
			"/",
		)

		return
	}

	prefixPathTmplStr = urlTmplStr[:idx]
	resourceTmplStr = slashDecoderRe.ReplaceAllLiteralString(
		urlTmplStr[idx+1:],
		"/",
	)

	return
}

func getHost(urlTmplStr string) (
	hostTmplStr string,
	secure, tslash bool,
	err error,
) {
	if urlTmplStr == "" {
		err = ErrEmptyHostTemplate
		return
	}

	var lurlTmplStr = len(urlTmplStr)
	urlTmplStr = strings.TrimPrefix(urlTmplStr, "https://")
	if len(urlTmplStr) < lurlTmplStr {
		secure = true
	} else {
		urlTmplStr = strings.TrimPrefix(urlTmplStr, "http://")
	}

	var idx = strings.IndexByte(urlTmplStr, '/')
	if idx < 0 {
		hostTmplStr = urlTmplStr
		return
	}

	hostTmplStr = urlTmplStr[:idx]
	urlTmplStr = urlTmplStr[idx:]
	if urlTmplStr == "/" {
		tslash = true
	} else {
		// Host template has a resource template.
		secure = false
		hostTmplStr = ""
		err = ErrUnwantedPathTemplate
	}

	return
}

func splitPathSegments(path string) (
	pss []string,
	root bool,
	tslash bool,
	err error,
) {
	if path == "/" {
		return []string{"/"}, true, false, nil
	}

	var psi = makePathSegmentIterator(path)
	for ps := psi.nextSegment(); ps != ""; ps = psi.nextSegment() {
		pss = append(pss, ps)
	}

	if psi.remainingPath() != "" {
		err = ErrEmptyPathSegmentTemplate
		return
	}

	return pss, false, psi.pathHasTrailingSlash(), nil
}

// responderURL returns the responder's URL with the URL values applied.
func responderURL(_r _Responder, hpVs HostPathValues) (*url.URL, error) {
	var (
		host string
		pss  []string
	)

loop:
	for p := _Parent(_r); p != nil; p = p.parent() {
		switch p := p.(type) {
		case *Resource:
			if p.IsRoot() {
				// Root "/" is added later when the segments are joined.
				continue
			}

			var tmpl = p.Template()
			if tmpl.IsStatic() {
				pss = append(pss, tmpl.Content())
				continue
			}

			var ps, err = tmpl.Apply(hpVs, false)
			if err != nil {
				return nil, newError("%w", err)
			}

			pss = append(pss, ps)
		case *Host:
			var tmpl = p.Template()
			if tmpl.IsStatic() {
				host = tmpl.Content()
			} else {
				var err error
				host, err = tmpl.Apply(hpVs, false)
				if err != nil {
					return nil, newError("%w", err)
				}
			}

			break loop
		}
	}

	var strb = strings.Builder{}
	for i := len(pss) - 1; i > -1; i-- {
		strb.WriteByte('/')
		strb.WriteString(pss[i])
	}

	switch rr := _r.(type) {
	case *Resource:
		if rr.IsSubtreeHandler() && !rr.IsRoot() {
			strb.WriteByte('/')
		}
	case *Host:
		if rr.IsSubtreeHandler() {
			strb.WriteByte('/')
		}
	}

	var scheme = "http"
	if _r.IsSecure() {
		scheme = "https"
	}

	return &url.URL{Scheme: scheme, Host: host, Path: strb.String()}, nil
}

// --------------------------------------------------

// HostPathValues is a slice of the host and path segment values.
// It does not include query values.
type HostPathValues = TemplateValues

// --------------------------------------------------

type _Arg struct {
	key, value interface{}
}

type _Args []_Arg

func (pargs *_Args) set(key, value interface{}) {
	var i, _ = pargs.get(key)
	if i < 0 {
		*pargs = append(*pargs, _Arg{key, value})
		return
	}

	(*pargs)[i].value = value
}

func (args _Args) get(key interface{}) (int, interface{}) {
	for i, largs := 0, len(args); i < largs; i++ {
		if args[i].key == key {
			return i, args[i].value
		}
	}

	return -1, nil
}

// --------------------------------------------------

// Args is created for each request when the host template contains a
// pattern or when the request's URL contains path segments. It's kept in the
// request's context.
type Args struct {
	url                   *url.URL
	cleanPath             string
	currentPathSegmentIdx int

	subtreeExists bool
	handled       bool

	hostPathValues HostPathValues
	_r             _Responder

	slc _Args
}

// -------------------------

// getArgs returns a context with _RoutingData that keeps the
// passed URL and the _Resource.
func getArgs(url *url.URL, _r _Responder) *Args {
	var args = getArgsFromThePool(url, _r)
	var tmpPath = url.Path

	// The escaped path may have a slash "/", which is a part of the path
	// segment, not a separator. So the unescaped path must be used for routing.
	//
	// As the documentation of the URL.EscapedPath() states, it may return a
	// different path from the URL.RawPath. Sometimes it's not suitable for
	// our intentions. It's preferable to use URL.RawPath if it's not empty.
	if len(url.RawPath) > 0 {
		tmpPath = url.RawPath
	}

	if tmpPath != "" {
		var trailingSlash bool
		var ltmpPath = len(tmpPath)
		if ltmpPath > 1 && tmpPath[ltmpPath-1] == '/' {
			trailingSlash = true
		}

		if tmpPath[0] != '/' {
			tmpPath = "/" + tmpPath
		}

		var cleanPath = path.Clean(tmpPath)
		if trailingSlash && len(cleanPath) > 1 {
			cleanPath += "/"
		}

		if len(cleanPath) != len(tmpPath) {
			args.cleanPath = cleanPath
		}
	}

	return args
}

// -------------------------

func (args *Args) pathIsRoot() bool {
	if args.cleanPath != "" {
		return args.cleanPath == "/"
	}

	if args.url.RawPath != "" {
		return args.url.RawPath == "/"
	}

	return args.url.Path == "/"
}

// pathLen returns the length of the path that rd is using.
func (args *Args) pathLen() int {
	var lpath = len(args.cleanPath)
	if lpath == 0 {
		lpath = len(args.url.RawPath)
		if lpath == 0 {
			lpath = len(args.url.Path)
		}
	}

	return lpath
}

// nextPathSegment returns the unescaped next path segment of the request's URL
// below the resource that is using the routing data.
func (args *Args) nextPathSegment() (string, error) {
	var lpath = args.pathLen()
	if args.currentPathSegmentIdx == lpath {
		return "", nil
	}

	if args.currentPathSegmentIdx == 0 {
		args.currentPathSegmentIdx++
		return "/", nil
	}

	var pathWasCleaned = args.cleanPath != ""
	var pathIsRaw = args.url.RawPath != ""

	var idx int
	if pathWasCleaned {
		idx = strings.IndexByte(
			args.cleanPath[args.currentPathSegmentIdx:],
			'/',
		)
	} else if pathIsRaw {
		idx = strings.IndexByte(
			args.url.RawPath[args.currentPathSegmentIdx:],
			'/',
		)
	} else {
		idx = strings.IndexByte(
			args.url.Path[args.currentPathSegmentIdx:],
			'/',
		)
	}

	var cIdx = args.currentPathSegmentIdx
	args.currentPathSegmentIdx += idx + 1
	idx += cIdx

	if idx < cIdx {
		idx = lpath
		args.currentPathSegmentIdx = lpath
	}

	if pathIsRaw {
		if pathWasCleaned {
			return url.PathUnescape(args.cleanPath[cIdx:idx])
		}

		return url.PathUnescape(args.url.RawPath[cIdx:idx])
	}

	if pathWasCleaned {
		return args.cleanPath[cIdx:idx], nil
	}

	return args.url.Path[cIdx:idx], nil
}

// reachedTheLastPathSegment returns true when the resource that is using the
// routing data is the last resource in the request's URL.
func (args *Args) reachedTheLastPathSegment() bool {
	return args.currentPathSegmentIdx == args.pathLen()
}

// pathHasTrailingSlash returns true if the request's URL has a trailing slash.
func (args *Args) pathHasTrailingSlash() bool {
	if lpath := len(args.cleanPath); lpath > 0 {
		return args.cleanPath != "/" && args.cleanPath[lpath-1] == '/'
	} else if lpath = len(args.url.RawPath); lpath > 0 {
		return args.url.RawPath != "/" && args.url.RawPath[lpath-1] == '/'
	}

	return args.url.Path != "" && args.url.Path != "/" &&
		args.url.Path[len(args.url.Path)-1] == '/'
}

// -------------------------

// HostPathValues returns the host and path values of the request's URL.
func (args *Args) HostPathValues() HostPathValues {
	return args.hostPathValues
}

// RemainingPath returns the escaped remaining path of the request's URL
// that's below the current responder's segment. The remaining path doesn't
// include the host, even when the method is called from the middleware of
// the router's segment handler.
func (rd *Args) RemainingPath() string {
	if rd.reachedTheLastPathSegment() {
		return ""
	}

	var pathWasCleaned = len(rd.cleanPath) > 0
	var pathIsRaw = len(rd.url.RawPath) > 0

	if rd._r != nil && rd._r.HasTrailingSlash() || rd.pathIsRoot() {
		// If the _r is a host or resource with a trailing slash, or if the
		// request's path contains only a slash "/" (root), the remaining path
		// should not start with or contain only a trailing slash. When the
		// request's path contains only a slash "/", that means the remaining
		// path is being retrieved by a host which is lenient on the trailing
		// slash or a root resource.
		if rd.currentPathSegmentIdx == 0 {
			if pathWasCleaned {
				return rd.cleanPath[1:]
			}

			if pathIsRaw {
				return rd.url.RawPath[1:]
			}

			return rd.url.Path[1:]
		}
	} else if rd.currentPathSegmentIdx > 0 {
		if pathWasCleaned {
			return rd.cleanPath[rd.currentPathSegmentIdx-1:]
		}

		if pathIsRaw {
			return rd.url.RawPath[rd.currentPathSegmentIdx-1:]
		}

		return rd.url.Path[rd.currentPathSegmentIdx-1:]
	}

	if pathWasCleaned {
		return rd.cleanPath[rd.currentPathSegmentIdx:]
	}

	if pathIsRaw {
		return rd.url.RawPath[rd.currentPathSegmentIdx:]
	}

	return rd.url.Path[rd.currentPathSegmentIdx:]
}

// ResponderSharedData returns the shared data of the host or resource that
// is currently handling the request. If the shared data wasn't set, nil is
// returned.
func (args *Args) ResponderSharedData() interface{} {
	if args._r == nil {
		return nil
	}

	return args._r.SharedData()
}

// ResponderImpl returns the implementation of the host or resource that
// is currently handling the request. If the host or resource wasn't created
// from an Impl or if they have no Impl set, nil is returned.
func (args *Args) ResponderImpl() Impl {
	if args._r == nil {
		return nil
	}

	return args._r.Implementation()
}

// Host returns the *Host of the request's URL. If there is no *Host, nil is
// returned.
func (args *Args) Host() *Host {
	switch _r := args._r.(type) {
	case *Host:
		return _r
	case *Resource:
		return _r.Host()
	default:
		return nil
	}
}

// CurrentResource returns the current *Resource that is handling the request.
// If the request is being handled by a host, nil is returned. In that
// case, the Host method must be used.
func (args *Args) CurrentResource() *Resource {
	if _r, ok := args._r.(*Resource); ok {
		return _r
	}

	return nil
}

// Set sets the custom argument that is passed between middlewares and/or
// handlers. The rules for defining a key are the same as in the context
// package. The key must be comparable and its type must be custom defined.
// The recommended type is an empty struct (struct{}) to avoid allocation
// when assigning to an interface{}. As stated in the context package,
// exported key variables' static type should be a pointer or interface.
func (args *Args) Set(key, value interface{}) {
	args.slc.set(key, value)
}

// Get returns the custom argument that was set with the Set method.
func (args *Args) Get(key interface{}) interface{} {
	var _, v = args.slc.get(key)
	return v
}

// -------------------------

type _ArgsKey struct{}

var ArgsKey interface{} = _ArgsKey{}

// ArgsFrom is a convenience function to retrieve *Args in the http.Handler or
// http.HandlerFunc after the conversion to the Handler or HandlerFunc with the
// Hr and HrFn functions.
func ArgsFrom(r *http.Request) *Args {
	var args, _ = r.Context().Value(ArgsKey).(*Args)
	return args
}

// --------------------------------------------------

var argsPool = sync.Pool{
	New: func() interface{} {
		return &Args{}
	},
}

func putArgsInThePool(args *Args) {
	args.cleanPath = ""
	args.currentPathSegmentIdx = 0

	args.subtreeExists = false
	args.handled = false

	if args.hostPathValues != nil {
		args.hostPathValues = args.hostPathValues[:0]
	}

	if args.slc != nil {
		args.slc = args.slc[:0]
	}

	// Other fields will be set at retrieval.

	argsPool.Put(args)
}

func getArgsFromThePool(url *url.URL, _r _Responder) *Args {
	var args = argsPool.Get().(*Args)
	args.url = url
	args._r = _r

	return args
}

// --------------------------------------------------

func newError(description string, args ...interface{}) error {
	if pc, _, _, ok := runtime.Caller(1); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			var strb strings.Builder
			strb.WriteString(fn.Name())
			strb.WriteString("() ")
			strb.WriteString(description)

			return fmt.Errorf(strb.String(), args...)
		}
	}

	return fmt.Errorf(description, args...)
}

// --------------------------------------------------

// traverseAndCall traverses all the _Resources in the passed _Resource trees
// and calls the f on each _Resource.
func traverseAndCall(rs []_Responder, f func(_Responder) error) error {
	type node struct {
		rs   []_Responder
		next *node
	}

	var (
		crs, irs []_Responder
		lcrs     int
		err      error
	)

	var n = &node{rs: rs}
	var currentN = n
	for currentN != nil {
		crs, lcrs = currentN.rs, len(currentN.rs)
		for i := 0; i < lcrs; i++ {
			err = f(crs[i])
			if err != nil {
				return err
			}

			irs = crs[i]._Resources()
			if irs != nil {
				n.next = &node{rs: irs}
				n = n.next
			}
		}

		currentN = currentN.next
	}

	return nil
}
