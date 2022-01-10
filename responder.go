// Copyright (c) 2021 Shohruh Adham
// Use of this source code is governed by the MIT License.

package nanomux

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ErrNilArgument is returned when one of the function arguments is nil.
var ErrNilArgument = fmt.Errorf("nil argument")

// ErrConflictingHost is returned when there is a conflict between a resource's
// host and its parent resource's host or a host in a URL template. Conflict
// can be the existence or absence of the host or a difference in a host
// template.
var ErrConflictingHost = fmt.Errorf("conflicting host")

// ErrConflictingPath is returned when there is a difference between a
// resource's prefix path and a prefix path in a URL template.
var ErrConflictingPath = fmt.Errorf("conflicting path")

// ErrConflictingPathSegment is returned when there is a difference between
// one of the resource's prefix path segments and a corresponding path segment
// in a URL template.
var ErrConflictingPathSegment = fmt.Errorf("conflicting path segment")

// ErrConflictingSecurity is returned when the argument URL template has a
// different scheme from the resource's scheme, or the resource is insecure
// (https is not required by the resource to respond), and the argument config
// has the RedirectInsecureRequest property set.
var ErrConflictingSecurity = fmt.Errorf("conflicting security")

// ErrConflictingTrailingSlash is returned when the argument URL template has a
// different trailing slash property than the one the resource was configured
// with.
var ErrConflictingTrailingSlash = fmt.Errorf("conflicting trailing slash")

// ErrConfictingConfig is returned when the argument config is different from
// the resource's configuration.
var ErrConflictingConfig = fmt.Errorf("conflicting config")

// ErrEmptyHostTemplate is returned when a host is required but its template is
// empty or the URL template doesn't contain a host template.
var ErrEmptyHostTemplate = fmt.Errorf("empty host template")

// ErrEmptyPathTemplate is returned when a path template is required but it's
// empty or a URL template doesn't contain a path template.
var ErrEmptyPathTemplate = fmt.Errorf("empty path template")

// ErrEmptyPathSegmentTemplate is returned when one of the path segment
// templates is empty in a path template.
var ErrEmptyPathSegmentTemplate = fmt.Errorf("empty path segment template")

// ErrWildcardHostTemplate is returned when a host template is a wildcard.
var ErrWildcardHostTemplate = fmt.Errorf("wildcard host template")

// ErrUnwantedPathTemplate is returned when a host template also contains a
// path template.
var ErrUnwantedPathTemplate = fmt.Errorf("unwanted path template")

// ErrNonRouterParent is returned on an attempt to register a host or a root
// resource under another host or resource.
var ErrNonRouterParent = fmt.Errorf("non-router parent")

// ErrUnnamedResource is returned when a resource has a pattern in its template
// or has a wildcard template, but has no name.
var ErrUnnamedResource = fmt.Errorf("unnamed resource")

// ErrDuplicateHostTemplate is returned when registering a new host if there
// is another host with the same template and both of them can handle a request.
var ErrDuplicateHostTemplate = fmt.Errorf("duplicate host template")

// ErrDuplicateResourceTemplate is returned when registering a new resource
// if there is another resource with the same template and both of them can
// handle a request.
var ErrDuplicateResourceTemplate = fmt.Errorf("duplicate resource template")

// ErrDuplicateNameInTheURL is returned when a new resource's name is not
// unique in its URL.
var ErrDuplicateNameInTheURL = fmt.Errorf("duplicate name in the URL")

// ErrDuplicateValueNameInTheURL is returned when one of the value names
// in the resource's template is a duplicate of a value name in the host's
// or another resource's template.
var ErrDuplicateValueNameInTheURL = fmt.Errorf(
	"duplicate value name in the URL",
)

// ErrDuplicateNameAmongSiblings is returned when a new resource's name is not
// unique among the resources registered under the same host or resource.
var ErrDuplicateNameAmongSiblings = fmt.Errorf("duplicate name among siblings")

// ErrDummyHost is returned when a host doesn't have a request handler for any
// HTTP method and an attempt to set a handler for the not allowed methods or
// to wrap one of the HTTP method handlers occurs.
var ErrDummyHost = fmt.Errorf("dummy host")

// ErrDummyResource is returned when a resource doesn't have a request handler
// for any HTTP method and an attempt to set a handler for the not allowed
// methods or to wrap one of the HTTP method handlers occurs.
var ErrDummyResource = fmt.Errorf("dummy resource")

// ErrRegisteredHost is returned on an attempt to register an already
// registered host. A host is considered registered even if it is registered
// under a different router.
var ErrRegisteredHost = fmt.Errorf("registered host")

// ErrRegisteredResource is returned on an attempt to register an already
// registered resource. A resource is considered registered even if it was
// registered under a different router, host, or resource.
var ErrRegisteredResource = fmt.Errorf("registered resource")

// ErrNonExistentHost is returned on an attempt to change the state of a
// non-existent host.
var ErrNonExistentHost = fmt.Errorf("non-existent host")

// ErrNonExistentResource is returned on an attempt to change the state of a
// non-existent resource.
var ErrNonExistentResource = fmt.Errorf("non-existent resource")

// --------------------------------------------------

// Config contains resource properties.
// The scheme and trailing slash properties are configured from the
// resource's URL. For example, if not configured differently,
// "https://example.com/resource/" means the resource ignores requests when
// the conection is not over "https", and redirects requests when their URL does
// not end with a trailing slash.
type Config struct {
	// SubtreeHandler means that a host or resource can handle a request
	// when there is no child resource with the matching template to handle
	// the request's next path segment. The remaining path is available in the
	// request's context and can be retrieved with the RemainingPathKey.
	SubtreeHandler bool

	// Secure means that a host or resource can be available only under https.
	Secure bool

	// RedirectInsecureRequest allows the resource to redirect the request from
	// an insecure endpoint to a secure one, i.e., from http to https, instead
	// of responding with a "404 Not Found" status code.
	RedirectInsecureRequest bool

	// TrailingSlash means that a host or resource has a trailing slash in
	// their URL. If a request is made to a URL without a trailing slash, the
	// host or resource redirects it to a URL with a trailing slash.
	TrailingSlash bool

	// StrictOnTrailingSlash tells the resource to drop the
	// request when the existence or absence of the trailing slash in the
	// request's URL doesn't match the resource's. By default, resources
	// redirect requests to the matching version of the URL.
	StrictOnTrailingSlash bool

	// LeniencyOnTrailingSlash allows the resource to respond, ignoring the
	// fact of the existence or absence of the trailing slash in the request's
	// URL. By default, resources redirect requests to the matching version of
	// the URL.
	LeniencyOnTrailingSlash bool

	// LeniencyOnUncleanPath allows the resource to respond, ignoring unclean
	// paths, i.e., paths with empty path segments or containing dots (relative
	// paths). By default, resources redirect requests to the clean version of
	// the URL.
	//
	// When used with a non-subtree host, the LeniencyOnUncleanPath property has
	// no effect.
	LeniencyOnUncleanPath bool

	// HandleThePathAsIs can be used to set both the LeniencyOnTrailingSlash
	// and the LeniencyOnUncleanPath at the same time.
	HandleThePathAsIs bool
}

// asFlags returns the Config properties set to true as an 8-bit _ConfigFlags.
func (config Config) asFlags() _ConfigFlags {
	var cfs _ConfigFlags
	if config.SubtreeHandler {
		cfs.set(flagSubtreeHandler)
	}

	if config.Secure {
		cfs.set(flagSecure)
	}

	if config.RedirectInsecureRequest {
		cfs.set(flagSecure | flagRedirectInsecure)
	}

	if config.TrailingSlash {
		cfs.set(flagTrailingSlash)
	}

	if config.StrictOnTrailingSlash {
		cfs.set(flagStrictOnTrailingSlash)
	}

	if config.LeniencyOnTrailingSlash {
		cfs.set(flagLeniencyOnTrailingSlash)
	}

	if config.LeniencyOnUncleanPath {
		cfs.set(flagLeniencyOnUncleanPath)
	}

	if config.HandleThePathAsIs {
		cfs.set(flagHandleThePathAsIs)
	}

	return cfs
}

// --------------------------------------------------

// _ConfigFlags keeps the resource properties as bit flags.
type _ConfigFlags uint8

const (
	flagActive _ConfigFlags = 1 << iota
	flagSubtreeHandler
	flagSecure
	flagRedirectInsecure
	flagTrailingSlash
	flagStrictOnTrailingSlash
	flagLeniencyOnTrailingSlash
	flagLeniencyOnUncleanPath
	flagHandleThePathAsIs = flagLeniencyOnTrailingSlash | flagLeniencyOnUncleanPath
)

func (cfs *_ConfigFlags) set(flags _ConfigFlags) {
	*cfs |= flags
}

func (cfs _ConfigFlags) has(flags _ConfigFlags) bool {
	return (cfs & flags) == flags
}

func (cfs _ConfigFlags) asConfig() Config {
	return Config{
		SubtreeHandler:          cfs.has(flagSubtreeHandler),
		Secure:                  cfs.has(flagSecure),
		RedirectInsecureRequest: cfs.has(flagRedirectInsecure),
		TrailingSlash:           cfs.has(flagTrailingSlash),
		StrictOnTrailingSlash:   cfs.has(flagStrictOnTrailingSlash),
		LeniencyOnTrailingSlash: cfs.has(flagLeniencyOnTrailingSlash),
		LeniencyOnUncleanPath:   cfs.has(flagLeniencyOnUncleanPath),
		HandleThePathAsIs:       cfs.has(flagHandleThePathAsIs),
	}
}

// --------------------------------------------------

// _Responder interface is the common interface between the Host and Resource
// interfaces.
type _Responder interface {
	Name() string
	Template() *Template
	URL(HostPathValues) (*url.URL, error)

	Router() *Router

	setParent(p _Parent) error
	parent() _Parent

	resourcesInThePath() []_Responder

	SetSharedData(data interface{})
	SharedData() interface{}

	setConfigFlags(flag _ConfigFlags)
	updateConfigFlags(cfs _ConfigFlags)
	configFlags() _ConfigFlags
	configCompatibility(secure, tslash bool, cfs *_ConfigFlags) error

	Configure(config Config)
	Config() Config

	IsSubtreeHandler() bool
	IsSecure() bool
	RedirectsInsecureRequest() bool
	HasTrailingSlash() bool
	IsStrictOnTrailingSlash() bool
	IsLenientOnTrailingSlash() bool
	IsLenientOnUncleanPath() bool
	HandlesThePathAsIs() bool

	// -------------------------

	canHandleRequest() bool

	checkNamesOfTheChildrenAreUniqueInTheURL(r *Resource) error
	validate(tmpl *Template) error
	validateHostTmpl(tmplStr string) error
	validateURL(hostTmplstr, pathTmplStr string) (
		remainingPathTmplStr string,
		err error,
	)

	resourceWithTemplate(tmpl *Template) (*Resource, error)
	registeredResource(pathTmplStr string) (r *Resource, tslash bool, err error)
	passChildResourcesTo(r _Responder) error
	registerResource(r *Resource) error
	segmentResources(pathSegments []string) (
		oldLast _Responder,
		newFirst, newLast *Resource,
		err error,
	)

	pathSegmentResources(path string) (
		oldLast _Responder,
		newFirst, newLast *Resource,
		tslash bool,
		err error,
	)

	registerResourceUnder(prefixPath string, r *Resource) error
	keepResourceOrItsChildResources(r *Resource) error

	Resource(path string) (*Resource, error)
	ResourceUsingConfig(path string, config Config) (*Resource, error)
	RegisterResource(r *Resource) error
	RegisterResourceUnder(prefixPath string, r *Resource) error
	RegisteredResource(path string) (*Resource, error)

	ChildResourceNamed(name string) *Resource
	ChildResources() []*Resource

	HasChildResource(r *Resource) bool
	HasAnyChildResources() bool

	// -------------------------

	SetImplementation(impl Impl) error
	Implementation() Impl

	SetHandlerFor(methods string, handler Handler) error
	SetHandlerFuncFor(methods string, handlerFunc HandlerFunc) error
	HandlerOf(method string) Handler

	WrapSegmentHandler(mwfs ...MiddlewareFunc) error
	WrapRequestHandler(mwfs ...MiddlewareFunc) error
	WrapHandlerOf(methods string, mwfs ...MiddlewareFunc) error

	// -------------------------

	ConfigurePath(path string, config Config) error
	PathConfig(path string) (Config, error)

	SetImplementationAt(path string, impl Impl) error
	ImplementationAt(path string) (Impl, error)

	SetPathHandlerFor(methods, path string, handler Handler) error
	SetPathHandlerFuncFor(methods, path string, handler HandlerFunc) error
	PathHandlerOf(method, path string) (Handler, error)

	WrapPathSegmentHandler(path string, mwfs ...MiddlewareFunc) error
	WrapPathRequestHandler(path string, mwfs ...MiddlewareFunc) error
	WrapPathHandlerOf(methods, path string, mwfs ...MiddlewareFunc) error

	// -------------------------

	ConfigureSubtree(config Config)

	WrapSubtreeSegmentHandlers(mwfs ...MiddlewareFunc) error
	WrapSubtreeRequestHandlers(mwfs ...MiddlewareFunc) error
	WrapSubtreeHandlersOf(methods string, mwfs ...MiddlewareFunc) error

	// -------------------------

	_Resources() []_Responder
	setRequestHandlerBase(rhb *_RequestHandlerBase)
	requestHandlerBase() *_RequestHandlerBase

	http.Handler
}

// --------------------------------------------------

// _ResponderBase implements the _Resource interface and provides the HostBase
// and ResourceBase types with common functionality.
type _ResponderBase struct {
	derived _Responder // Keeps the reference to the embedding struct.
	impl    Impl
	tmpl    *Template
	papa    _Parent

	staticResources  map[string]*Resource
	patternResources []*Resource
	wildcardResource *Resource

	*_RequestHandlerBase
	segmentHandler HandlerFunc
	requestHandler HandlerFunc

	cfs        _ConfigFlags
	sharedData interface{}
}

// --------------------------------------------------

// Name returns the name of the resource given in the resource's path
// segment template.
func (rb *_ResponderBase) Name() string {
	return rb.tmpl.Name()
}

// Template returns the path segment template of the resource.
func (rb *_ResponderBase) Template() *Template {
	return rb.tmpl
}

// URL returns the resource's URL with values applied to it.
func (rb *_ResponderBase) URL(values HostPathValues) (*url.URL, error) {
	var url, err = responderURL(rb.derived, values)
	if err != nil {
		return nil, newErr("%w", err)
	}

	return url, nil
}

// Router returns the router of the resource. The resource can be a host or a
// path segment resource. It's not required to be directly registered in the
// router.
func (rb *_ResponderBase) Router() *Router {
	for p := rb.papa; p != nil; p = p.parent() {
		if ro, ok := p.(*Router); ok {
			return ro
		}
	}

	return nil
}

// -------------------------

// setParent sets the resource's parent when it's being registered.
func (rb *_ResponderBase) setParent(p _Parent) error {
	if p == nil {
		rb.papa = nil
		return nil
	}

	if _, ok := rb.derived.(*Host); ok {
		// Only a router can be set as a parent for a host.
		if _, ok := p.(*Router); !ok {
			return newErr("%w", ErrNonRouterParent)
		}
	}

	if rb.Template().UnescapedContent() == "/" {
		// Only a router can be set as a parent for a root.
		if _, ok := p.(*Router); !ok {
			return newErr("%w", ErrNonRouterParent)
		}
	}

	rb.papa = p
	return nil
}

// parent returns the resource's parent.
func (rb *_ResponderBase) parent() _Parent {
	return rb.papa
}

// resourcesInThePath returns all the resources above in the hierarchy
// (including a host and the resource itself).
func (rb *_ResponderBase) resourcesInThePath() []_Responder {
	var resources []_Responder
	for p := rb.derived.(_Parent); p != nil; p = p.parent() {
		if _, ok := p.(*Router); ok {
			break
		}

		resources = append(resources, p.(_Responder))
	}

	var lresources = len(resources)
	for i, k := 0, lresources-1; i < k; i, k = i+1, k-1 {
		resources[i], resources[k] = resources[k], resources[i]
	}

	return resources
}

// -------------------------

// SetSharedData sets the data that is shared between the request handlers.
// It's useful when the handlers are not the resource's own methods.
//
// Shared data can be retrieved through the request's context by calling its
// Value method with the ResourcesSharedDataKey. If the shared data can be
// modified, accessing it must be synchronized with a mutex or some other
// synchronization method.
//
// Example:
//
// 	type SharedData struct {
//		*sync.Mutex // Must be initilized.
//		X SomeType
// 	}
//
// 	...
//
// 	func SomeHandler(w http.ResponseWriter, r *http.Request) {
// 		var sharedData = r.Context().Value(ResourcesSharedDataKey)
//		sharedData.Lock()
//		defer sharedData.Unlock()
//		sharedData.X = someValue
//		...
// 	}
func (rb *_ResponderBase) SetSharedData(data interface{}) {
	rb.sharedData = data
}

// SharedData returns the data set by SetSharedData.
func (rb *_ResponderBase) SharedData() interface{} {
	return rb.sharedData
}

// -------------------------

// setConfigFlags is used to add config flags.
func (rb *_ResponderBase) setConfigFlags(flag _ConfigFlags) {
	rb.cfs.set(flag)
}

// updateConfigFlags is used to update existing config flags to the passed
// config flags.
func (rb *_ResponderBase) updateConfigFlags(cfs _ConfigFlags) {
	rb.cfs = cfs
}

// configFlags returns the resource's config flags.
func (rb *_ResponderBase) configFlags() _ConfigFlags {
	return rb.cfs
}

// configCompatibility checks the configured resource's properties for
// compatibility with the arguments. If the resource wasn't configured,
// the function configures it with the arguments. If the cfs parameter is
// nil, it's ignored.
func (rb *_ResponderBase) configCompatibility(
	secure, tslash bool,
	cfs *_ConfigFlags,
) error {
	var rbcfs = rb.configFlags()
	if rbcfs.has(flagActive) {
		if rbcfs.has(flagSecure) != secure {
			return newErr("%w", ErrConflictingSecurity)
		}

		if !rbcfs.has(flagLeniencyOnTrailingSlash) && rbcfs.has(flagTrailingSlash) != tslash {
			return newErr("%w", ErrConflictingTrailingSlash)
		}

		if cfs != nil {
			if !rbcfs.has(*cfs) {
				return newErr("%w", ErrConflictingConfig)
			}
		}
	} else {
		rb.setConfigFlags(flagActive)

		if secure {
			rb.setConfigFlags(flagSecure)
		}

		if tslash {
			rb.setConfigFlags(flagTrailingSlash)
		}

		if cfs != nil {
			rb.setConfigFlags(*cfs)
		}
	}

	return nil
}

// Configure configures the host or resource with config.
// If the host or resource has been configured before, it's reconfigured.
func (rb *_ResponderBase) Configure(config Config) {
	rb.updateConfigFlags(flagActive | config.asFlags())
}

// Config returns the configuration of the host or resource.
func (rb *_ResponderBase) Config() Config {
	return rb.cfs.asConfig()
}

// IsSubtreeHandler returns true if the resource was configured as a subtree.
func (rb *_ResponderBase) IsSubtreeHandler() bool {
	return rb.cfs.has(flagSubtreeHandler)
}

// IsSecure returns true if the resource was configured to respond only if
// it is used under "https".
func (rb *_ResponderBase) IsSecure() bool {
	return rb.cfs.has(flagSecure)
}

// RedirectsInsecureRequest returns true if the resource was configured to
// redirect insecure requests, instead of responding with a "404 Not Found"
// status code.
//
// The resource can be configured to redirect insecure requests if it's
// intended to be used in both "http" and "https" servers.
func (rb *_ResponderBase) RedirectsInsecureRequest() bool {
	return rb.cfs.has(flagRedirectInsecure)
}

// HasTrailingSlash returns true if the resource's URL ends with a trailing
// slash. If the resource has a trailing slash in its URL and the request is
// made to the URL without the trailing slash, the resource redirects it to its
// URL with the trailing slash and vice versa.
func (rb *_ResponderBase) HasTrailingSlash() bool {
	return rb.cfs.has(flagTrailingSlash)
}

// IsStrictOnTrailingSlash returns true if the resource was configured to
// drop the request when the existence or absence of the trailing slash in
// the request's URL doesn't match with its own URL. By default, the resource
// redirects the request on unmatched trailing slash.
func (rb *_ResponderBase) IsStrictOnTrailingSlash() bool {
	return rb.cfs.has(flagStrictOnTrailingSlash)
}

// IsLenientOnTrailingSlash returns true if the resource was configured to
// ignore an unmatched trailing slash in the request's URL.
func (rb *_ResponderBase) IsLenientOnTrailingSlash() bool {
	return rb.cfs.has(flagLeniencyOnTrailingSlash)
}

// IsLenientOnUncleanPath returns true if the resource was configured to ignore
// unclean paths like "example.com///.//resource1//resource2".
func (rb *_ResponderBase) IsLenientOnUncleanPath() bool {
	return rb.cfs.has(flagLeniencyOnUncleanPath)
}

// HandlesThePathAsIs returns true if the resource was configured to be lenient
// on both, trailing slash and unclean paths.
func (rb *_ResponderBase) HandlesThePathAsIs() bool {
	return rb.cfs.has(flagHandleThePathAsIs)
}

// canHandleRequest returns true if the resource has any HTTP method handler.
func (rb *_ResponderBase) canHandleRequest() bool {
	return rb._RequestHandlerBase != nil &&
		len(rb._RequestHandlerBase.mhPairs) > 0
}

// -------------------------

// checkNamesAreUniqueInTheURL checks whether the name and value names of
// the template are unique in the resource's URL.
func (rb *_ResponderBase) checkNamesAreUniqueInTheURL(tmpl *Template) error {
	var tmplValueNames = tmpl.ValueNames()
	if tmpl.name == "" && tmplValueNames == nil {
		return nil
	}

	for p := _Parent(rb); p != nil; p = p.parent() {
		if r, ok := p.(_Responder); ok {
			if tmpl.name != "" && r.Name() == tmpl.name {
				return ErrDuplicateNameInTheURL
			}

			if r.Template().HasValueName(tmplValueNames...) {
				return ErrDuplicateValueNameInTheURL
			}
		} else {
			break
		}
	}

	return nil
}

// checkNamesOfTheChildrenAreUniqueInTheURL checks whether the child resources
// of the argument resource have unique names above in the receiver resource's
// hierarchy.
func (rb *_ResponderBase) checkNamesOfTheChildrenAreUniqueInTheURL(
	r *Resource,
) error {
	if _, ok := rb.derived.(*Host); ok {
		return nil
	}

	for _, chr := range r.ChildResources() {
		var err = rb.checkNamesAreUniqueInTheURL(chr.Template())
		if err != nil {
			return err
		}

		err = rb.checkNamesOfTheChildrenAreUniqueInTheURL(chr)
		if err != nil {
			return err
		}
	}

	return nil
}

// validate checks whether the argument template pointer is nil or a non-static
// template without a name. It also checks the name of a non-static template
// for uniqueness above in the resource's hierarchy.
func (rb *_ResponderBase) validate(tmpl *Template) error {
	if tmpl == nil {
		return newErr("%w", ErrNilArgument)
	}

	if err := rb.checkNamesAreUniqueInTheURL(tmpl); err != nil {
		return newErr("%w", err)
	}

	return nil
}

// validateHostTmpl checks whether the argument template is the template of the
// resource's host. Validation fails even if the resource doesn't have a host.
func (rb *_ResponderBase) validateHostTmpl(tmplStr string) error {
	if tmplStr != "" {
		var h *Host
		switch _r := rb.derived.(type) {
		case *Host:
			h = _r
		case *Resource:
			h = _r.Host()
		}

		if h == nil {
			return newErr("%w", ErrConflictingHost)
		}

		var tmpl, err = TryToParse(tmplStr)
		if err != nil {
			return newErr("%w", err)
		}

		if tmpl.IsWildcard() {
			return newErr("%w", ErrWildcardHostTemplate)
		}

		if err = h.Template().SimilarityWith(tmpl).Err(); err != nil {
			return newErr("%w", err)
		}
	}

	return nil
}

// validateURL checks whether the argument host and path templates are the
// templates of the host, prefix path segment resources, and the resource
// itself. The method also returns the remaining part of the path template
// string below the resource.
func (rb *_ResponderBase) validateURL(hostTmplStr string, pathTmplStr string) (
	remainingPathTmplStr string,
	err error,
) {
	var resources = rb.resourcesInThePath()
	if err := resources[0].validateHostTmpl(hostTmplStr); err != nil {
		return "", newErr("%w", err)
	}

	var lresources = len(resources)
	if _, ok := resources[0].(*Host); ok {
		if lresources == 1 {
			if pathTmplStr == "" || pathTmplStr == "/" {
				return "", nil
			}
		}

		resources = resources[1:]
		lresources = len(resources)
	}

	var psi = makePathSegmentIterator(pathTmplStr)
	for i := 0; i < lresources; i++ {
		var ps = psi.nextSegment()
		if ps == "" {
			return "", newErr("%w", ErrConflictingPath)
		}

		var tmpl, err = TryToParse(ps)
		if err != nil {
			return "", newErr("%w", err)
		}

		var rtmpl = resources[i].Template()
		var similarity = rtmpl.SimilarityWith(tmpl)
		if similarity != TheSame {
			return "", newErr("%w %q", ErrConflictingPathSegment, ps)
		}
	}

	remainingPathTmplStr = psi.remainingPath()
	return
}

// resourceWithTemplate returns the existing child resource with a similar
// template to the argument.
func (rb *_ResponderBase) resourceWithTemplate(tmpl *Template) (
	*Resource,
	error,
) {
	if tmpl.IsStatic() {
		var r = rb.staticResources[tmpl.UnescapedContent()]
		if r != nil {
			var stmpl = r.Template()
			if stmpl == tmpl {
				return r, nil
			}

			if stmpl.Name() != tmpl.Name() {
				return nil, newErr("%w", ErrDifferentNames)
			}

			return r, nil
		}
	} else if tmpl.IsWildcard() {
		if rb.wildcardResource != nil {
			var wtmpl = rb.wildcardResource.Template()
			if wtmpl == tmpl {
				return rb.wildcardResource, nil
			}

			switch sim := wtmpl.SimilarityWith(tmpl); sim {
			case DifferentValueNames:
				fallthrough
			case DifferentNames:
				return nil, newErr("%w", sim.Err())
			case TheSame:
				return rb.wildcardResource, nil
			}
		}
	} else {
		for _, pr := range rb.patternResources {
			var ptmpl = pr.Template()
			if ptmpl == tmpl {
				return pr, nil
			}

			switch sim := ptmpl.SimilarityWith(tmpl); sim {
			case DifferentValueNames:
				fallthrough
			case DifferentNames:
				return nil, newErr("%w", sim.Err())
			case TheSame:
				return pr, nil
			}
		}
	}

	return nil, nil
}

// registeredResource returns the child resource below in the hierarchy if it
// can be reached with the path template.
//
// Unlike other methods, registeredResoure accepts a path template string that
// doesn't have a full template string for each path segment resource. If the
// path segment resource has a name, it can be used instead of the full
// template string.
//
// For example:
//		/childResourceTemplate/$someName/anotherTemplate/$anotherName
// 		/$someChildResourceName/$anotherResourceName
func (rb *_ResponderBase) registeredResource(
	pathTmplStr string,
) (r *Resource, tslash bool, err error) {
	var _r _Responder = rb
	var psi = makePathSegmentIterator(pathTmplStr)

	for ps := psi.nextSegment(); ps != ""; ps = psi.nextSegment() {
		var (
			name, tmplStr string
			tmpl          *Template
		)

		name, tmplStr, err = templateNameAndContent(ps)
		if tmplStr == "" {
			if name == "" {
				return nil, false, ErrEmptyPathSegmentTemplate
			}

			r = rb.ChildResourceNamed(name)
		} else {
			tmpl, err = TryToParse(ps)
			if err != nil {
				return
			}

			r, err = _r.resourceWithTemplate(tmpl)
			if err != nil {
				return
			}
		}

		if r == nil {
			return
		}

		_r = r
	}

	if psi.remainingPath() != "" {
		return nil, false, newErr("%w", ErrEmptyPathSegmentTemplate)
	}

	return r, psi.pathHasTrailingSlash(), nil
}

// passChildResourcesTo method transfers all of the child resources to the
// argument resource.
func (rb *_ResponderBase) passChildResourcesTo(r _Responder) error {
	for _, rr := range rb.staticResources {
		if err := r.keepResourceOrItsChildResources(rr); err != nil {
			return newErr("%w", err)
		}
	}

	for _, rr := range rb.patternResources {
		if err := r.keepResourceOrItsChildResources(rr); err != nil {
			return newErr("%w", err)
		}
	}

	if rb.wildcardResource != nil {
		err := r.keepResourceOrItsChildResources(rb.wildcardResource)
		if err != nil {
			return newErr("%w", err)
		}
	}

	rb.staticResources = nil
	rb.patternResources = nil
	rb.wildcardResource = nil

	return nil
}

// replaceResource replaces the old child resource with the new one. The method
// doesn't compare the templates of the resources. It assumes they are the same.
func (rb *_ResponderBase) replaceResource(oldR, newR *Resource) error {
	var tmpl = oldR.Template()
	switch {
	case tmpl.IsStatic():
		rb.staticResources[tmpl.UnescapedContent()] = newR
	case tmpl.IsWildcard():
		rb.wildcardResource = newR
	default:
		var idx = -1
		for i, r := range rb.patternResources {
			if r == oldR {
				idx = i
				break
			}
		}

		rb.patternResources[idx] = newR
	}

	var err = newR.setParent(rb.derived)
	if err != nil {
		return newErr("%w", err)
	}

	err = oldR.setParent(nil)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// registerResource registers the argument resource and sets the receiver
// resource as its parent.
func (rb *_ResponderBase) registerResource(r *Resource) error {
	switch tmpl := r.Template(); {
	case tmpl.IsStatic():
		if rb.staticResources == nil {
			rb.staticResources = make(map[string]*Resource)
		}

		rb.staticResources[tmpl.UnescapedContent()] = r
	case tmpl.IsWildcard():
		rb.wildcardResource = r
	default:
		rb.patternResources = append(rb.patternResources, r)
	}

	var err = r.setParent(rb.derived)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// segmentResources finds or creates and returns the resources below in the
// hierarchy using the argument path segment templates. Newly created resources
// will be registered one under the other in the order given in the argument
// slice. But they won't be registered under the last existing resource. It's
// the responsibility of the caller.
func (rb *_ResponderBase) segmentResources(pathSegments []string) (
	oldLast _Responder,
	newFirst, newLast *Resource,
	err error,
) {
	oldLast = rb.derived
	for lpathSegments, i := len(pathSegments), 0; i < lpathSegments; i++ {
		var tmpl *Template
		tmpl, err = TryToParse(pathSegments[i])
		if err != nil {
			err = newErr("path segment %s %w", pathSegments[i], err)
			return
		}

		var r *Resource
		if newFirst == nil {
			r, err = oldLast.resourceWithTemplate(tmpl)
			if err != nil {
				err = newErr("path segment %s %w", pathSegments[i], err)
				return
			}
		}

		if r != nil {
			oldLast = r
		} else {
			if err = oldLast.validate(tmpl); err != nil {
				err = newErr("path segment %s %w", pathSegments[i], err)
				return
			}

			var r = newDummyResource(tmpl)
			if newLast != nil {
				err = newLast.checkNamesAreUniqueInTheURL(tmpl)
				if err != nil {
					err = newErr("%w", err)
					return
				}

				if err = newLast.registerResource(r); err != nil {
					err = newErr("%w", err)
					return
				}
			} else {
				newFirst = r
			}

			newLast = r
		}
	}

	return
}

// pathSegmentResources finds or creates and returns the resources below
// in the hierarchy using the argument path template. Newly created resources
// will be registered one under the other in the order given in the path
// template string. But they won't be registered under the last existing
// resource. It's the responsibility of the caller.
func (rb *_ResponderBase) pathSegmentResources(pathTmplStr string) (
	oldLast _Responder,
	newFirst, newLast *Resource,
	tslash bool,
	err error,
) {
	var root bool
	var pss []string
	pss, root, tslash, err = splitPathSegments(pathTmplStr)
	if err != nil {
		return
	}

	if root {
		if _, ok := rb.derived.(*Host); ok {
			oldLast = rb
			return
		}

		err = newErr("%w", ErrNonRouterParent)
		return
	}

	oldLast, newFirst, newLast, err = rb.segmentResources(pss)
	if err != nil {
		tslash = false
	}

	return
}

// registerResourceUnder registeres the argument resource below in the hierarchy
// of the receiver resource under the given prefix path segments. It also
// creates and registers the prefix path segments below in the hierarchy, if
// they don't exist.
func (rb *_ResponderBase) registerResourceUnder(
	prefixPath string,
	r *Resource,
) error {
	var oldLast, newFirst, newLast, _, err = rb.pathSegmentResources(prefixPath)
	if err != nil {
		return err
	}

	if newFirst != nil {
		if err := newLast.checkNamesOfTheChildrenAreUniqueInTheURL(r); err != nil {
			return newErr("%w", err)
		}

		if r := oldLast.ChildResourceNamed(newFirst.Name()); r != nil {
			return newErr("%w", ErrDuplicateNameAmongSiblings)
		}

		if err = newLast.registerResource(r); err != nil {
			return newErr("%w", err)
		}

		if err = oldLast.registerResource(newFirst); err != nil {
			return newErr("%w", err)
		}

		return nil
	}

	if err := oldLast.checkNamesOfTheChildrenAreUniqueInTheURL(r); err != nil {
		return newErr("%w", err)
	}

	err = oldLast.keepResourceOrItsChildResources(r)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// keepResourceOrItsChildResources is intended to be used when there is a
// template collision between resources. In that scenario, the function keeps
// one of them depending on whether one has request handlers set. The function
// also passes the child resources of the resource that cannot handle a request
// to the one that can. If both resources can handle a request, then the
// ErrDuplicateResourceTemplate error will be returned.
func (rb *_ResponderBase) keepResourceOrItsChildResources(r *Resource) error {
	var rwt, err = rb.resourceWithTemplate(r.Template())
	if err != nil {
		return newErr("%w", err)
	}

	if rwt == nil {
		if err = rb.registerResource(r); err != nil {
			return newErr("%w", err)
		}

		return nil
	}

	// CHECK: Maybe we mustn't compare the flagActive.
	var rcfs = r.configFlags()
	err = rwt.configCompatibility(
		rcfs.has(flagSecure),
		rcfs.has(flagTrailingSlash),
		&rcfs,
	)

	if err != nil {
		return newErr("%w", err)
	}

	if !r.canHandleRequest() {
		err = r.passChildResourcesTo(rwt)
		if err != nil {
			return newErr("%w", err)
		}

		return nil
	}

	if !rwt.canHandleRequest() {
		err = rwt.passChildResourcesTo(r)
		if err != nil {
			return newErr("%w", err)
		}

		err = rb.replaceResource(rwt, r)
		if err != nil {
			return newErr("%w", err)
		}

		return nil
	}

	return newErr(
		"%w %s",
		ErrDuplicateResourceTemplate,
		rwt.Template().String(),
	)
}

// Resource uses the path template to find an existing resource or to create
// a new one below in the hierarchy of the receiver resource and returns it.
// If the path template contains prefix segments that don't exist, the method
// also creates new resources for them.
//
// If the resource exists, its scheme and trailing slash properties are
// compared to the values given in the path template. If there is a difference,
// the method returns an error. If the method creates a new resource, its
// scheme and trailing slash properties are configured using the values given
// within the path template.
//
// The names given to the path segment resources must be unique in the path and
// among their respective siblings.
func (rb *_ResponderBase) Resource(path string) (*Resource, error) {
	var (
		hTmplStr       string
		secure, tslash bool
		err            error
	)

	hTmplStr, path, secure, tslash, err = splitHostAndPath(path)
	if err != nil {
		return nil, newErr("%w", err)
	}

	if hTmplStr != "" {
		return nil, newErr("%w", ErrNonRouterParent)
	}

	if path == "" {
		return nil, newErr("%w", ErrEmptyPathTemplate)
	}

	if path[0] != '/' {
		path = "/" + path
	}

	var oldLast _Responder
	var newFirst, newLast *Resource
	oldLast, newFirst, newLast, _, err = rb.pathSegmentResources(path)
	if err != nil {
		return nil, newErr("%w", err)
	}

	if newFirst != nil {
		err = newLast.configCompatibility(secure, tslash, nil)
		if err != nil {
			return nil, newErr("%w", err)
		}

		if oldLast.ChildResourceNamed(newFirst.Name()) != nil {
			return nil, newErr("%w", ErrDuplicateNameAmongSiblings)
		}

		if err = oldLast.registerResource(newFirst); err != nil {
			return nil, newErr("%w", err)
		}

		return newLast, nil
	}

	err = oldLast.configCompatibility(secure, tslash, nil)
	if err != nil {
		return nil, newErr("%w", err)
	}

	return oldLast.(*Resource), nil
}

// ResourceUsingConfig uses the path template and config to find an existing
// resource or to create a new one below in the hierarchy of the receiver
// resource and returns it. If the path template contains prefix segments
// that don't exist, the method also creates new resources for them.
//
// If the resource exists, its configuration is compared to the argument config.
// Also, its scheme and trailing slash properties are compared to the values
// given in the path template. If there is a difference, the function returns
// an error. If the function creates a new resource, it's configured using the
// config and the values given in the path template.
//
// The names of the path segment resources must be unique within the path and
// among their respective siblings.
func (rb *_ResponderBase) ResourceUsingConfig(
	pathTmplStr string,
	config Config,
) (*Resource, error) {
	var (
		hTmplStr       string
		secure, tslash bool
		err            error
	)

	hTmplStr, pathTmplStr, secure, tslash, err = splitHostAndPath(pathTmplStr)
	if err != nil {
		return nil, newErr("%w", err)
	}

	if hTmplStr != "" {
		return nil, newErr("%w", ErrNonRouterParent)
	}

	if pathTmplStr == "" {
		return nil, newErr("%w", ErrEmptyPathTemplate)
	}

	if config.RedirectInsecureRequest && !secure {
		return nil, newErr("%w", ErrConflictingSecurity)
	}

	if pathTmplStr[0] != '/' {
		pathTmplStr = "/" + pathTmplStr
	}

	var oldLast _Responder
	var newFirst, newLast *Resource
	oldLast, newFirst, newLast, _, err = rb.pathSegmentResources(pathTmplStr)
	if err != nil {
		return nil, newErr("%w", err)
	}

	var cfs = config.asFlags()
	if newFirst != nil {
		err = newLast.configCompatibility(secure, tslash, &cfs)
		if err != nil {
			return nil, newErr("%w", err)
		}

		if r := oldLast.ChildResourceNamed(newFirst.Name()); r != nil {
			return nil, newErr("%w", ErrDuplicateNameAmongSiblings)
		}

		if err = oldLast.registerResource(newFirst); err != nil {
			return nil, newErr("%w", err)
		}

		return newLast, nil
	}

	err = oldLast.configCompatibility(secure, tslash, &cfs)
	if err != nil {
		return nil, newErr("%w", err)
	}

	return oldLast.(*Resource), nil
}

// RegisterResource registers the argument resource below in the hierarchy of
// the receiver resource.
//
// If the argument resource has a URL template, its corresponding host and path
// segments must be compatible with the templates of the host and path segment
// resources above in the hierarchy. The remaining path segments are used as the
// prefix segments for the argument resource below the receiver resource. If
// there are compatible resources with the remaining path segments below the
// receiver resource, the argument resource will be registered under them.
// Otherwise, new resources will be created for the missing path segments.
//
// If the argument resource's template collides with one of its siblings'
// templates, RegisterResource checks which one has the request handlers set
// and passes the other one's child resources to it. If both can handle a
// request, the method returns an error. Child resources are also checked
// recursively.
func (rb *_ResponderBase) RegisterResource(r *Resource) error {
	if r == nil {
		return newErr("%w", ErrNilArgument)
	}

	if r.IsRoot() {
		return newErr("%w", ErrNonRouterParent)
	}

	if r.parent() != nil {
		return newErr("%w", ErrRegisteredResource)
	}

	if err := rb.validate(r.Template()); err != nil {
		return newErr("%w", err)
	}

	if err := rb.checkNamesOfTheChildrenAreUniqueInTheURL(r); err != nil {
		return newErr("%w", err)
	}

	if urlt := r.urlTmpl(); urlt != nil {
		var rppss, err = rb.validateURL(urlt.Host, urlt.PrefixPath)
		if err != nil {
			return newErr("%w", err)
		}

		if len(rppss) > 0 {
			err = rb.registerResourceUnder(rppss, r)
			if err != nil {
				return newErr("%w", err)
			}

			return nil
		}
	}

	if err := rb.keepResourceOrItsChildResources(r); err != nil {
		return newErr("%w", err)
	}

	return nil
}

// RegisterResourceUnder registers the argument resource below the receiver
// resource and the prefix path segments.
//
// If the argument resource has a URL template, its host and path segment
// templates must be compatible with the corresponding host and path segment
// resources in the hierarchy and with the argument prefix path segments.
// If there are existing resources compatible with the prefix path segments,
// the argument resource will be registered under them, otherwise new resources
// will be created for the missing segments.
//
// If the prefix path segment resources exist and the argument resource's
// template collides with the last prefix resource's child resource,
// RegisterResourceUnder checks which one has the request handlers set and
// passes the other one's child resources to it. If both can handle a request,
// the method returns an error.
//
// The trailing slash in the prefix path is ignored.
func (rb *_ResponderBase) RegisterResourceUnder(
	prefixPath string,
	r *Resource,
) error {
	if r == nil {
		return newErr("%w", ErrNilArgument)
	}

	if r.IsRoot() {
		return newErr("%w", ErrNonRouterParent)
	}

	if r.parent() != nil {
		return newErr("%w", ErrRegisteredResource)
	}

	if err := rb.validate(r.Template()); err != nil {
		return newErr("%w", err)
	}

	if err := rb.checkNamesOfTheChildrenAreUniqueInTheURL(r); err != nil {
		return newErr("%w", err)
	}

	if prefixPath == "/" {
		if _, ok := rb.derived.(*Host); ok {
			prefixPath = ""
		} else {
			return newErr("%w", ErrNonRouterParent)
		}
	}

	if prefixPath != "" && prefixPath[0] != '/' {
		prefixPath = "/" + prefixPath
	}

	if urlt := r.urlTmpl(); urlt != nil {
		if urlt.PrefixPath != "" {
			var lpp, lurltPp = len(prefixPath), len(urlt.PrefixPath)
			if lpp > 0 {
				if lastIdx := lpp - 1; prefixPath[lastIdx] == '/' {
					prefixPath = prefixPath[:lastIdx]
					lpp--
				}
			}

			if lpp > lurltPp {
				return newErr("%w", ErrConflictingPath)
			}

			var pp = urlt.PrefixPath
			if strings.HasSuffix(urlt.PrefixPath, prefixPath) {
				pp = urlt.PrefixPath[:lurltPp-lpp]
			}

			var rppss, err = rb.validateURL(urlt.Host, pp)
			if err != nil {
				return newErr("%w", err)
			}

			if len(rppss) > 0 {
				return newErr("%w", ErrConflictingPath)
			}
		} else {
			if err := rb.validateHostTmpl(urlt.Host); err != nil {
				return newErr("%w", err)
			}
		}
	}

	if prefixPath != "" {
		var err = rb.registerResourceUnder(prefixPath, r)
		if err != nil {
			return newErr("%w", err)
		}

		return nil
	}

	if err := rb.keepResourceOrItsChildResources(r); err != nil {
		return newErr("%w", err)
	}

	return nil
}

// RegisteredResource returns the resource in the hierarchy below the receiver
// resource if it can be reached with the path template. In the path template,
// names can be used instead of the complete segment templates.
//
// For example,
// 		/childResourceTemplate/$someName/anotherResourceTemplate/,
//		https:///$childResourceName/$grandChildResourceName
//
// The scheme and trailing slash properties must be compatible with the
// resource's otherwise the method returns an error.
func (rb *_ResponderBase) RegisteredResource(pathTmplStr string) (
	*Resource,
	error,
) {
	var (
		hTmplStr       string
		secure, tslash bool
		err            error
	)

	hTmplStr, pathTmplStr, secure, tslash, err = splitHostAndPath(pathTmplStr)
	if err != nil {
		return nil, newErr("%w", err)
	}

	if hTmplStr != "" {
		return nil, newErr("%w", ErrNonRouterParent)
	}

	if pathTmplStr == "" {
		return nil, newErr("%w", ErrEmptyPathTemplate)
	}

	if pathTmplStr == "/" {
		return nil, newErr("%w", ErrNonRouterParent)
	}

	var r *Resource
	r, _, err = rb.registeredResource(pathTmplStr)
	if err != nil {
		return nil, newErr("%w", err)
	}

	if r != nil {
		err = r.configCompatibility(secure, tslash, nil)
		if err != nil {
			return nil, err
		}

		return r, nil
	}

	return nil, nil
}

// ChildResourceNamed returns the named resource if it exists, otherwise it
// returns nil. Only the direct child resources of the receiver resource will
// be looked at.
func (rb *_ResponderBase) ChildResourceNamed(name string) *Resource {
	if name == "" {
		return nil
	}

	if rb.wildcardResource != nil && rb.wildcardResource.Name() == name {
		return rb.wildcardResource
	}

	for _, r := range rb.patternResources {
		if r.Name() == name {
			return r
		}
	}

	for _, r := range rb.staticResources {
		if r.Name() == name {
			return r
		}
	}

	return nil
}

// ChildResources returns all the child resources of the receiver resource.
// If the receiver resource doesn't have any child resources, the method
// returns nil.
func (rb *_ResponderBase) ChildResources() []*Resource {
	var rs []*Resource
	for _, r := range rb.staticResources {
		rs = append(rs, r)
	}

	rs = append(rs, rb.patternResources...)

	if rb.wildcardResource != nil {
		rs = append(rs, rb.wildcardResource)
	}

	return rs
}

// HasChildResource returns true if the argument resource is a direct child
// of the receiver resource.
func (rb *_ResponderBase) HasChildResource(r *Resource) bool {
	if r == nil {
		return false
	}

	var tmpl = r.Template()
	if tmpl.IsStatic() {
		for _, sr := range rb.staticResources {
			if sr == r {
				return true
			}
		}
	} else if tmpl.IsWildcard() && r == rb.wildcardResource {
		return true
	} else {
		for _, pr := range rb.patternResources {
			if pr == r {
				return true
			}
		}
	}

	return false
}

// HasAnyChildResources returns true if the receiver resource has any child
// resources.
func (rb *_ResponderBase) HasAnyChildResources() bool {
	if len(rb.staticResources) > 0 || len(rb.patternResources) > 0 ||
		rb.wildcardResource != nil {
		return true
	}

	return false
}

// -------------------------

// SetImplementation sets the request handlers from the passed impl.
// The impl is also kept for future retrieval. All existing handlers
// are discarded.
func (rb *_ResponderBase) SetImplementation(impl Impl) error {
	if impl == nil {
		return newErr("%w", ErrNilArgument)
	}

	var rhb, err = detectHTTPMethodHandlersOf(impl)
	if err != nil {
		return newErr("%w", err)
	}

	rb.impl = impl

	if rhb != nil {
		rb.setRequestHandlerBase(rhb)
	}

	return nil
}

// Implementation returns the implementation of the host or resource.
// If the host or resource wasn't created from an Impl or if they have no
// Impl set, nil is returned.
func (rb *_ResponderBase) Implementation() Impl {
	return rb.impl
}

// -------------------------

// SetHandlerFor sets the handler as a request handler for HTTP methods.
//
// The argument methods is a case-insensitive list of HTTP methods separated
// by a comma and/or space. An exclamation mark "!" denotes the handler of the
// not allowed HTTP methods and must be used alone. Which means that setting the
// not allowed HTTP methods' handler must happen in a separate call. Examples of
// methods: "get", "PUT POST", "get, custom" or "!".
func (rb *_ResponderBase) SetHandlerFor(
	methods string,
	handler Handler,
) error {
	if rb._RequestHandlerBase == nil {
		var rhb = &_RequestHandlerBase{}
		var err = rhb.setHandlerFor(methods, handler)
		if err != nil {
			if errors.Is(err, ErrNoHandlerExists) {
				if _, ok := rb.derived.(*Host); ok {
					return newErr("%w %s", ErrDummyHost, err)
				}

				return newErr("%w %s", ErrDummyResource, err)
			}

			return newErr("%w", err)
		}

		rb.setRequestHandlerBase(rhb)
	} else {
		var err = rb.setHandlerFor(methods, handler)
		if err != nil {
			return newErr("%w", err)
		}
	}
	return nil
}

// SetHandlerFuncFor sets the handler function as a request handler for the
// HTTP methods.
//
// The argument methods is a case-insensitive list of HTTP methods separated
// by a comma and/or space. An exclamation mark "!" denotes the handler of the
// not allowed HTTP methods and must be used alone. Which means that setting the
// not allowed HTTP methods' handler must happen in a separate call. Examples of
// methods: "get", "PUT POST", "get, custom" or "!".
func (rb *_ResponderBase) SetHandlerFuncFor(
	methods string,
	handlerFunc HandlerFunc,
) error {
	var err = rb.SetHandlerFor(methods, handlerFunc)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// HandlerOf returns the HTTP method's handler of the resource. If the
// resource or the handler, doesn't exist, nil is returned.
//
// The argument method is an HTTP method. An exclamation mark "!" can be used
// to get the handler of HTTP methods that are not allowed. Examples: "get",
// "POST" or "!".
func (rb *_ResponderBase) HandlerOf(method string) Handler {
	if rb._RequestHandlerBase == nil {
		return nil
	}

	return rb.handlerOf(method)
}

// -------------------------

// WrapSegmentHandler wraps the resource's segment handler with the middlewares
// in their passed order.
//
// The segment handler is called when the request passes through the resource.
// It calls the request handler of its own resource if the resource is the last
// resource in the request's URL. Or, it finds the next resource that matches
// the next path segment and passes the request to it. If there is no matching
// resource for the next path segment, the handler for a not-found resource is
// called. The host's segment handler calls the request handler if the request
// was made to the host.
func (rb *_ResponderBase) WrapSegmentHandler(mwfs ...MiddlewareFunc) error {
	if len(mwfs) == 0 {
		return newErr("%w", ErrNoMiddleware)
	}

	for i, mw := range mwfs {
		if mw == nil {
			return newErr("%w at index %d", ErrNilArgument, i)
		}

		rb.segmentHandler = mw(rb.segmentHandler)
	}

	return nil
}

// WrapRequestHandler wraps the resource's request handler with the middlewares
// in their passed order.
//
// The request handler calls the HTTP method handler of the resource depending
// on the request's method. Unlike the segment handler, the request handler is
// called only when the resource is going to handle the request.
func (rb *_ResponderBase) WrapRequestHandler(mwfs ...MiddlewareFunc) error {
	if len(mwfs) == 0 {
		return newErr("%w", ErrNoMiddleware)
	}

	if !rb.canHandleRequest() {
		if _, ok := rb.derived.(*Host); ok {
			return newErr("%w", ErrDummyHost)
		}

		return newErr("%w", ErrDummyResource)
	}

	for i, mw := range mwfs {
		if mw == nil {
			return newErr("%w at index %d", ErrNilArgument, i)
		}

		rb.requestHandler = mw(rb.requestHandler)
	}

	return nil
}

// WrapHandlerOf wraps the handler of the HTTP methods with the middlewares in
// their passed order. If the handler doesn't exist for any given method, the
// method returns an error.
//
// The argument methods is a case-insensitive list of HTTP methods separated
// by a comma and/or space. An exclamation mark "!" denotes the handler of the
// not allowed HTTP methods, and an asterisk "*" denotes all the handlers of
// HTTP methods in use. Both must be used alone. Which means that wrapping the
// not allowed HTTP methods' handler and all handlers of HTTP methods in use
// must happen in separate calls. Examples of methods: "get", "PUT POST", "get,
// custom", "*" or "!".
func (rb *_ResponderBase) WrapHandlerOf(
	methods string,
	mwfs ...MiddlewareFunc,
) error {
	if rb._RequestHandlerBase == nil {
		if _, ok := rb.derived.(*Host); ok {
			return newErr("%w", ErrDummyHost)
		}

		return newErr("%w", ErrDummyResource)
	}

	var err = rb.wrapHandlerOf(methods, mwfs...)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// -------------------------

// ConfigurePath configures the existing resource at the path. If the resource
// was configured before, it will be reconfigured.
func (rb *_ResponderBase) ConfigurePath(path string, config Config) error {
	var r, err = rb.RegisteredResource(path)
	if err != nil {
		return newErr("%w", err)
	}

	if r == nil {
		return newErr("%w", ErrNonExistentResource)
	}

	r.Configure(config)
	return nil
}

// PathConfig returns the configuration of the existing resource.
func (rb *_ResponderBase) PathConfig(path string) (Config, error) {
	var r, err = rb.RegisteredResource(path)
	if err != nil {
		return Config{}, newErr("%w", err)
	}

	if r == nil {
		return Config{}, newErr("%w", ErrNonExistentResource)
	}

	return r.Config(), nil
}

// -------------------------

// SetImplementationAt sets the request handlers for a resource at the path
// from the passed Impl. If the resource doesn't exist, the method creates it.
// The resource keeps the impl for future retrieval. Existing handlers of the
// resource are discarded.
//
// The scheme and trailing slash property values in the path template must be
// compatible with the existing resource's properties, otherwise the function
// returns an error. A newly created resource is configured with the values in
// the path template.
func (rb *_ResponderBase) SetImplementationAt(
	path string,
	rh Impl,
) error {
	var r, err = rb.Resource(path)
	if err != nil {
		return newErr("%w", err)
	}

	err = r.SetImplementation(rh)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// ImplementationAt returns the implementation of the resource at the path.
// If the resource doesn't exist or it wasn't created from an Impl or it has
// no Impl set, nil is returned.
//
// The scheme and trailing slash property values in the path template must be
// compatible with the resource's properties, otherwise the method returns an
// error.
func (rb *_ResponderBase) ImplementationAt(path string) (Impl, error) {
	var r, err = rb.RegisteredResource(path)
	if err != nil {
		return nil, newErr("%w", err)
	}

	if r == nil {
		return nil, newErr("%w", ErrNonExistentResource)
	}

	return r.Implementation(), nil
}

// -------------------------

// SetPathHandlerFor sets the HTTP methods' handler for a resource at the path.
// If the resource doesn't exist, it will be created.
//
// The scheme and trailing slash property values in the path template must be
// compatible with the existing resource's properties, otherwise the function
// returns an error. A newly created resource is configured with the values in
// the path template.
//
// The argument methods is a case-insensitive list of HTTP methods separated
// by a comma and/or space. An exclamation mark "!" denotes the handler of the
// not allowed HTTP methods and must be used alone. Which means that setting the
// not allowed HTTP methods' handler must happen in a separate call. Examples of
// methods: "get", "PUT POST", "get, custom" or "!".
func (rb *_ResponderBase) SetPathHandlerFor(
	methods, path string,
	handler Handler,
) error {
	var r, err = rb.Resource(path)
	if err != nil {
		return newErr("%w", err)
	}

	err = r.SetHandlerFor(methods, handler)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// SetPathHandlerFuncFor sets the HTTP methods' handler function for a
// resource at the path. If the resource doesn't exist, it will be created.
//
// The scheme and trailing slash property values in the path template must be
// compatible with the existing resource's properties, otherwise the function
// returns an error. A newly created resource is configured with the values in
// the path template.
//
// The argument methods is a case-insensitive list of HTTP methods separated
// by a comma and/or space. An exclamation mark "!" denotes the handler of the
// not allowed HTTP methods and must be used alone. Which means that setting the
// not allowed HTTP methods' handler must happen in a separate call. Examples of
// methods: "get", "PUT POST", "get, custom" or "!".
func (rb *_ResponderBase) SetPathHandlerFuncFor(
	methods, path string,
	handler HandlerFunc,
) error {
	var r, err = rb.Resource(path)
	if err != nil {
		return newErr("%w", err)
	}

	err = r.SetHandlerFor(methods, handler)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// PathHandlerOf returns the HTTP method's handler of the resource at the path.
// If the resource doesn't exist, nil is returned.
//
// The scheme and trailing slash property values in the path template must be
// compatible with the resource's properties, otherwise the function returns
// an error.
//
// The argument method is an HTTP method. An exclamation mark "!" can be used
// to get the handler of HTTP methods that are not allowed. Examples: "get",
// "POST" or "!".
func (rb *_ResponderBase) PathHandlerOf(method, path string) (
	Handler,
	error,
) {
	var r, err = rb.RegisteredResource(path)
	if err != nil {
		return nil, newErr("%w", err)
	}

	if r == nil {
		return nil, newErr("%w", ErrNonExistentResource)
	}

	return r.HandlerOf(method), nil
}

// WrapPathSegmentHandler wraps the segment handler of the resource at the path.
// Handler is wrapped in the middlewares' passed order. If the resource doesn't
// exist, an error is returned.
//
// The segment handler is called when the request passes through the resource.
// It calls the request handler of its own resource if the resource is the last
// resource in the request's URL. Or, it finds the next resource that matches
// the next path segment and passes the request to it. If there is no matching
// resource for the next path segment, the handler for a not-found resource is
// called. The host's segment handler calls the request handler if the request
// was made to the host.
//
// The scheme and trailing slash property values in the URL template must be
// compatible with the resource's properties, otherwise the method returns an
// error.
func (rb *_ResponderBase) WrapPathSegmentHandler(
	path string,
	mwfs ...MiddlewareFunc,
) error {
	var r, err = rb.RegisteredResource(path)
	if err != nil {
		return newErr("%w", err)
	}

	if r == nil {
		return newErr("%w", ErrNonExistentResource)
	}

	err = r.WrapSegmentHandler(mwfs...)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// WrapPathRequestHandler wraps the request handler of the resource at the path.
// Handler is wrapped in the middlewares' passed order. If the resource doesn't
// exist, an error is returned.
//
// The request handler calls the HTTP method handler of the resource depending
// on the request's method. Unlike the segment handler, the request handler is
// called only when the resource is going to handle the request.
//
// The scheme and trailing slash property values in the URL template must be
// compatible with the resource's properties, otherwise the method returns an
// error.
func (rb *_ResponderBase) WrapPathRequestHandler(
	path string,
	mwfs ...MiddlewareFunc,
) error {
	var r, err = rb.RegisteredResource(path)
	if err != nil {
		return newErr("%w", err)
	}

	if r == nil {
		return newErr("%w", ErrNonExistentResource)
	}

	err = r.WrapRequestHandler(mwfs...)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// WrapPathHandlerOf wraps the handlers of HTTP methods of the resource at the
// path. Handlers are wrapped in the middlewares' passed order.
//
// The argument methods is a case-insensitive list of HTTP methods separated
// by a comma and/or space. An exclamation mark "!" denotes the handler of the
// not allowed HTTP methods, and an asterisk "*" denotes all the handlers of
// HTTP methods in use. Both must be used alone. Which means that wrapping the
// not allowed HTTP methods' handler and all handlers of HTTP methods in use
// must happen in separate calls. Examples of methods: "get", "PUT POST", "get,
// custom", "*" or "!".
//
// If the resource or the handler of any HTTP method doesn't exist, the method
// returns an error.
func (rb *_ResponderBase) WrapPathHandlerOf(
	methods, path string,
	mwfs ...MiddlewareFunc,
) error {
	var r, err = rb.RegisteredResource(path)
	if err != nil {
		return newErr("%w", err)
	}

	if r == nil {
		return newErr("%w", err)
	}

	err = r.WrapHandlerOf(methods, mwfs...)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// -------------------------

// ConfigureSubtree configures all the resources below in the hierarchy.
func (rb *_ResponderBase) ConfigureSubtree(config Config) {
	traverseAndCall(
		rb._Resources(),
		func(_r _Responder) error {
			_r.Configure(config)
			return nil
		},
	)
}

// WrapSubtreeSegmentHandlers wraps the segment handlers of the resources
// in the hierarchy below the receiver resource. Handlers are wrapped in the
// middlewares' passed order.
//
// The segment handler is called when the request passes through the resource.
// It calls the request handler of its own resource if the resource is the last
// resource in the request's URL. Or, it finds the next resource that matches
// the next path segment and passes the request to it. If there is no matching
// resource for the next path segment, the handler for a not-found resource is
// called. The host's segment handler calls the request handler if the request
// was made to the host.
func (rb *_ResponderBase) WrapSubtreeSegmentHandlers(
	mwfs ...MiddlewareFunc,
) error {
	var err = traverseAndCall(
		rb._Resources(),
		func(_r _Responder) error {
			return _r.WrapSegmentHandler(mwfs...)
		},
	)

	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// WrapSubtreeRequestHandlers wraps the request handlers of the resources
// in the hierarchy below the receiver resource. Handlers are wrapped in the
// middlewares' passed order.
//
// The request handler calls the HTTP method handler of the resource depending
// on the request's method. Unlike the segment handler, the request handler is
// called only when the resource is going to handle the request.
func (rb *_ResponderBase) WrapSubtreeRequestHandlers(
	mwfs ...MiddlewareFunc,
) error {
	var err = traverseAndCall(
		rb._Resources(),
		func(_r _Responder) error {
			var err = _r.WrapRequestHandler(mwfs...)
			// Subtree below hosts cannot return the ErrDummyHost.
			// It's enough to check the ErrDummyResource.
			if errors.Is(err, ErrDummyResource) {
				return nil
			}

			return err
		},
	)

	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// WrapSubtreeHandlersOf wraps the HTTP method handlers of the resources in
// the hierarchy below the receiver resource.
//
// The argument methods is a case-insensitive list of HTTP methods separated
// by a comma and/or space. An exclamation mark "!" denotes the handler of the
// not allowed HTTP methods, and an asterisk "*" denotes all the handlers of
// HTTP methods in use. Both must be used alone. Which means that wrapping the
// not allowed HTTP methods' handler and all handlers of HTTP methods in use
// must happen in separate calls. Examples of methods: "get", "PUT POST", "get,
// custom", "*" or "!".
func (rb *_ResponderBase) WrapSubtreeHandlersOf(
	methods string,
	mwfs ...MiddlewareFunc,
) error {
	var err = wrapEveryHandlerOf(methods, rb._Resources(), mwfs...)
	if err != nil {
		return newErr("%w", err)
	}

	return nil
}

// -------------------------

// _Resources returns all the direct child resources.
func (rb *_ResponderBase) _Resources() []_Responder {
	var rhs []_Responder
	for _, rh := range rb.ChildResources() {
		rhs = append(rhs, rh)
	}

	return rhs
}

func (rb *_ResponderBase) setRequestHandlerBase(rhb *_RequestHandlerBase) {
	rb._RequestHandlerBase = rhb
	rb.requestHandler = HandlerFunc(rhb.handleRequest)
}

func (rb *_ResponderBase) requestHandlerBase() *_RequestHandlerBase {
	return rb._RequestHandlerBase
}

// -------------------------

// passRequestToChildResource passes the request that was made to a resource
// below in the hierarchy.
func (rb *_ResponderBase) passRequestToChildResource(
	w http.ResponseWriter,
	r *http.Request,
	args *Args,
) bool {
	var currentPathSegmentIdx = args.currentPathSegmentIdx
	var ps, err = args.nextPathSegment()
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)

		args.handled = true
		args.currentPathSegmentIdx = currentPathSegmentIdx
		return args.handled
	}

	if len(ps) > 0 {
		if sr := rb.staticResources[ps]; sr != nil {
			args._r = sr.derived
			sr.segmentHandler(w, r, args)

			args.currentPathSegmentIdx = currentPathSegmentIdx
			return args.handled
		}

		for _, pr := range rb.patternResources {
			var matched bool
			matched, args.hostPathValues = pr.Template().Match(
				ps,
				args.hostPathValues,
			)

			if matched {
				args._r = pr.derived
				pr.segmentHandler(w, r, args)

				args.currentPathSegmentIdx = currentPathSegmentIdx
				return args.handled
			}
		}

		if rb.wildcardResource != nil {
			_, args.hostPathValues = rb.wildcardResource.Template().Match(
				ps,
				args.hostPathValues,
			)

			args._r = rb.wildcardResource.derived
			rb.wildcardResource.segmentHandler(w, r, args)

			args.currentPathSegmentIdx = currentPathSegmentIdx
			return args.handled
		}
	}

	if args.subtreeExists {
		args.currentPathSegmentIdx = currentPathSegmentIdx
		return false
	}

	notFoundResourceHandler.ServeHTTP(w, r, args)
	args.handled = true
	args.currentPathSegmentIdx = currentPathSegmentIdx
	return true
}