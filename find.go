package main

import (
	"net/url"
	"strings"
	"time"

	"github.com/zellyn/kooky"
	_ "github.com/zellyn/kooky/browser/all"
)

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func findCookies(url *url.URL, name string, browsers []string, logger *Logger) (results []*kooky.Cookie) {
	logger.Printf("Looking for browsers %v", browsers)
	logger.Printf("Looking for cookies for URL %s", url)
	filter := currentlyAppliesToURLAndName(url, name, logger.RequireVerbosity(2))

	c := make(chan []*kooky.Cookie)
	cs := kooky.FindAllCookieStores()
	logger.Printf("Found %d stores", len(cs))

	done := make(chan struct{})

	// append cookies
	go func() {
		for cookies := range c {
			results = append(results, cookies...)
		}
		close(done)
	}()

	// read cookies
	for _, store := range cs {
		defer store.Close()

		if !contains(browsers, store.Browser()) {
			logger.Printf("Skipping %s", store.Browser())
			continue
		}
		logger.Printf("Found %s", store.Browser())

		cookies, err := store.ReadCookies(filter)
		if err == nil && cookies != nil {
			c <- cookies
		}
	}
	close(c)

	logger.Printf("Found %d matching cookie(s)", len(results))
	return
}

func currentlyAppliesToURLAndName(url *url.URL, name string, logger *Logger) kooky.Filter {
	currentTime := time.Now()
	logger.Printf("Current time is %v", currentTime)
	return appliesToURLAndNameAtTime(url, name, currentTime, logger)
}

func appliesToURLAndNameAtTime(url *url.URL, name string, time time.Time, logger *Logger) kooky.Filter {
	urlIsNotSecure := url.Scheme != "https"
	return kooky.FilterFunc(func(cookie *kooky.Cookie) bool {
		if !hostMatchesDomain(url.Host, cookie.Domain) {
			logger.Printf("Rejecting cookie for non-matching domain: %v", cookie)
		} else if urlIsNotSecure && cookie.Secure {
			logger.Printf("Rejecting secure cookie for non-HTTPS URL: %v", cookie)
		} else if !(cookie.Expires.IsZero() || time.Before(cookie.Expires)) {
			logger.Printf("Rejecting expired cookie: %v", cookie)
		} else if url.Path != "" && !strings.HasPrefix(url.Path, cookie.Path) {
			logger.Printf("Rejecting cookie due to unmatched path: %v", cookie)
		} else if name != "" && cookie.Name != name {
			logger.Printf("Rejecting cookie due to unmatched name: %v", cookie)
		} else {
			logger.Printf("Accepting: %v", cookie)
			return true
		}
		return false
	})
}

func hostMatchesDomain(host string, domain string) bool {
	return host == domain ||
		(strings.HasPrefix(domain, ".") && (strings.HasSuffix(host, domain) || host == domain[1:]))
}
