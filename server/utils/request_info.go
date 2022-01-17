package utils

import "net/http"

// GetIP helps in getting the IP address from the request
func GetIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}

	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}

// GetUserAgent helps in getting the user agent from the request
func GetUserAgent(r *http.Request) string {
	return r.UserAgent()
}
