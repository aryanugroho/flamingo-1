package redirects

import (
	"errors"
	"net/http"

	"log"

	"flamingo.me/flamingo/core/redirects/infrastructure"
	"flamingo.me/flamingo/framework/flamingo"
	"flamingo.me/flamingo/framework/router"
	"flamingo.me/flamingo/framework/web"
	"flamingo.me/flamingo/framework/web/responder"
)

type (
	redirector struct {
		responder.RedirectAware
		responder.ErrorAware
		logger flamingo.Logger
	}
)

var redirectDataMap map[string]infrastructure.CsvContent

func init() {
	redirectData := infrastructure.GetRedirectData()

	redirectDataMap = make(map[string]infrastructure.CsvContent)

	for i := range redirectData {
		redirectDataMap[redirectData[i].OriginalPath] = redirectData[i]
	}

	for _, entry := range redirectDataMap {
		foundEntry, err := findEntryInRedirectMap(entry.RedirectTarget)
		if err == nil {
			log.Printf("ERROR: found a chained redirect for %#v to %#v   Rejecting redirects because of loop risk", entry, foundEntry)
			redirectDataMap = nil
		}
	}

}

func (r *redirector) Inject(redirectAware responder.RedirectAware, errorAware responder.ErrorAware, logger flamingo.Logger) {
	r.RedirectAware = redirectAware
	r.ErrorAware = errorAware
	r.logger = logger
}

//TryServeHTTP - implementation of OptionalHandler (from prefixrouter package)
func (r *redirector) TryServeHTTP(rw http.ResponseWriter, req *http.Request) (bool, error) {
	contextPath := req.RequestURI
	//r.Logger.Debug("TryServeHTTP called with %v", contextPath)
	status, location, err := r.processRedirects(contextPath)
	if err != nil {
		return true, errors.New("no redirect found")
	}
	if location != "" {
		rw.Header().Set("Location", location)
	}
	rw.WriteHeader(status)
	return false, nil
}

//Filter - implementation of Filter interface (from router package)
func (r *redirector) Filter(ctx web.Context, w http.ResponseWriter, chain *router.FilterChain) web.Response {

	contextPath := ctx.Request().RequestURI

	status, location, err := r.processRedirects(contextPath)
	if err != nil {
		return chain.Next(ctx, w)
	}

	switch code := status; code {
	case http.StatusMovedPermanently:
		return r.RedirectPermanentURL(location)
	case http.StatusFound:
		return r.RedirectURL(location)
	case http.StatusGone:
		return r.ErrorAware.ErrorWithCode(ctx, errors.New("page is gone"), http.StatusGone)
	case http.StatusNotFound:
		return r.ErrorAware.ErrorNotFound(ctx, errors.New("page not found"))
	}

	return chain.Next(ctx, w)
}

//processRedirects - if a redirect is existing for given contextPath it returns the desired HTTP Response status and location (if relevant for the status) - if nothing is found it return 0
func (r *redirector) processRedirects(contextPath string) (status int, location string, error error) {

	entry, err := findEntryInRedirectMap(contextPath)
	if err != nil {
		//nothing found for contextPath
		return 0, "", errors.New("contextPath not found")
	}

	r.logger.Debug("Redirecting from %s to %s by %d", entry.OriginalPath, entry.RedirectTarget, entry.HTTPStatusCode)

	switch code := entry.HTTPStatusCode; code {
	case http.StatusMovedPermanently, http.StatusFound:
		return code, entry.RedirectTarget, nil
	case http.StatusGone:
		return http.StatusGone, "", nil
	default:
		//unsupported status - return 404 status
		return 404, "", nil
	}

}

func findEntryInRedirectMap(contextPath string) (*infrastructure.CsvContent, error) {
	if redirectDataMap == nil {
		return nil, errors.New("no redirects loaded")
	}
	if currentRedirect, ok := redirectDataMap[contextPath]; ok {
		return &currentRedirect, nil
	}
	return nil, errors.New("not found")
}
