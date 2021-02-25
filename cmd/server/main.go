package main

import (
	"flag"
	"io"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	fListenAddr string
	fProxy      string
	fOutputDir  string
	fLoglevel   string

	rootCmd = &cobra.Command{}
)

func main() {
	flag.StringVar(&fLoglevel, "l", "info", "")
	flag.StringVar(&fListenAddr, "p", ":9090", "")
	flag.StringVar(&fOutputDir, "o", "", "")
	flag.Parse()

	if lvl, err := logrus.ParseLevel(fLoglevel); err != nil {
		logrus.Panic("failed to parse log level: ", err)
	} else {
		logrus.SetLevel(lvl)
	}

	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logrus.Error(err)
			}
		}()

		// build url
		targetURL, err := url.Parse(r.URL.String())
		if err != nil {
			logrus.Warn("invalid url from request", err)
			return
		}
		if origHost := r.Header.Get("X-FORWARDED-HOST"); origHost != "" {
			targetURL.Host = origHost
		} else {
			logrus.Debug("not a proxied request, abort")
			return
		}
		if targetURL.Host == "" {
			targetURL.Host = r.Host
		}
		targetURL.Scheme = "https"

		// TODO: rewrite url

		// TODO: filter out invalid request

		// TOOD: write request to file

		// build request
		targetRequest, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
		if err != nil {
			logrus.Warn("failed to create request", err)
			return
		}

		targetRequest.Header = r.Header.Clone()
		for _, cookie := range r.Cookies() {
			targetRequest.AddCookie(cookie)
		}

		// do request
		targetResponse, err := http.DefaultClient.Do(targetRequest)
		if err != nil {
			logrus.Warn("failed to send request", err)
			return
		}

		// TODO: write response to file

		// proxy to sender
		for key := range targetResponse.Header {
			for _, value := range targetResponse.Header[key] {
				rw.Header().Set(key, value)
			}
		}
		for _, cookie := range targetResponse.Cookies() {
			http.SetCookie(rw, cookie)
		}
		rw.WriteHeader(targetResponse.StatusCode)
		if targetResponse.Body != nil {
			io.Copy(rw, targetResponse.Body)
			targetResponse.Body.Close()
		}
	})

	http.ListenAndServe(fListenAddr, handler)
}
