package v1

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/html"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1pb "github.com/yourselfhosted/slash/proto/gen/api/v1"
)

const (
	// maxResponseSize is the maximum response size in bytes (10MB).
	maxResponseSize = 10 * 1024 * 1024
	// defaultFetchTimeout is the default timeout for fetching a URL.
	defaultFetchTimeout = 10 * time.Second
	// maxRedirects is the maximum number of redirects to follow.
	maxRedirects = 5
)

// GetURLMetadata fetches social metadata from a URL.
func (*APIV1Service) GetURLMetadata(ctx context.Context, request *v1pb.GetURLMetadataRequest) (*v1pb.GetURLMetadataResponse, error) {
	// Validate URL
	parsedURL, err := url.Parse(request.Url)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid URL: %v", err)
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, status.Errorf(codes.InvalidArgument, "only http and https URLs are allowed")
	}

	// Block private IP ranges to prevent internal network access
	if parsedURL.Host != "" {
		hostname := parsedURL.Hostname()
		if hostname != "" {
			ip := net.ParseIP(hostname)
			// Check if IP is nil or is a private/reserved IP
			if ip != nil && (isPrivateIP(ip) || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || len(ip) == 0 || ip[0] == 0) {
				return nil, status.Errorf(codes.PermissionDenied, "access to private or reserved IP addresses is not allowed")
			}
		}
	}

	// Fetch the URL
	resp, err := fetchURLWithRedirects(ctx, request.Url, defaultFetchTimeout)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if !isSuccessStatusCode(resp.StatusCode) {
		return nil, status.Errorf(codes.InvalidArgument, "URL returned error status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		// Still try to parse if it's HTML-like
		if !strings.Contains(contentType, "html") {
			return nil, status.Errorf(codes.InvalidArgument, "URL does not point to an HTML document (content-type: %s)", contentType)
		}
	}

	// Read response body with size limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read response body: %v", err)
	}

	// Parse HTML and extract metadata
	ogMetadata := parseHTMLMetadata(body, resp.Request.URL.String())

	// Get favicon
	favicon := extractFavicon(resp.Request.URL.String(), body)

	// Get final URL after redirects
	finalURL := resp.Request.URL.String()

	return &v1pb.GetURLMetadataResponse{
		Title:       ogMetadata.title,
		Description: ogMetadata.description,
		Image:       ogMetadata.image,
		SiteName:    ogMetadata.siteName,
		Url:         finalURL,
		Favicon:     favicon,
	}, nil
}

// isSuccessStatusCode checks if the status code is a success (2xx).
func isSuccessStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

// isPrivateIP checks if an IP is a private IP address.
func isPrivateIP(ip net.IP) bool {
	// Check for IPv4 private ranges
	if ip.To4() != nil {
		return ip[0] == 10 ||
			(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
			(ip[0] == 192 && ip[1] == 168)
	}
	// Check for IPv6 private ranges (fc00::/7)
	return ip[0] == 0xfc || ip[0] == 0xfd
}

// fetchURLWithRedirects fetches a URL following redirects with configurable timeout.
func fetchURLWithRedirects(ctx context.Context, targetURL string, timeout time.Duration) (*http.Response, error) {
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	// Set user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SlashBot/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		// Check if it's a timeout error
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			return nil, status.Errorf(codes.DeadlineExceeded, "request timed out after %v: %s", timeout, err.Error())
		}
		return nil, err
	}

	return resp, nil
}

// ogMetadata holds extracted Open Graph metadata from HTML.
type ogMetadata struct {
	title       string
	description string
	image       string
	siteName    string
}

// parseHTMLMetadata extracts metadata from HTML content.
func parseHTMLMetadata(body []byte, baseURL string) ogMetadata {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return ogMetadata{}
	}

	m := ogMetadata{}
	var parseNode func(*html.Node)
	parseNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Check for meta tags
			if n.Data == "meta" {
				props := make(map[string]string)
				for _, attr := range n.Attr {
					props[attr.Key] = attr.Val
				}

				// Open Graph tags
				if ogTitle := props["property"]; ogTitle == "og:title" {
					m.title = props["content"]
				}
				if ogDesc := props["property"]; ogDesc == "og:description" {
					m.description = props["content"]
				}
				if ogImage := props["property"]; ogImage == "og:image" {
					m.image = resolveURL(props["content"], baseURL)
				}
				if ogSiteName := props["property"]; ogSiteName == "og:site_name" {
					m.siteName = props["content"]
				}

				// Twitter Card tags
				if twitterTitle := props["name"]; twitterTitle == "twitter:title" && m.title == "" {
					m.title = props["content"]
				}
				if twitterDesc := props["name"]; twitterDesc == "twitter:description" && m.description == "" {
					m.description = props["content"]
				}
				if twitterImage := props["name"]; twitterImage == "twitter:image" && m.image == "" {
					m.image = resolveURL(props["content"], baseURL)
				}

				// Fallback to basic meta tags
				if name := props["name"]; name == "description" && m.description == "" {
					m.description = props["content"]
				}
				if name := props["name"]; name == "title" && m.title == "" {
					m.title = props["content"]
				}
			}

			// Check for title tag
			if n.Data == "title" && m.title == "" {
				if n.FirstChild != nil {
					m.title = n.FirstChild.Data
				}
			}
		}

		// Recurse into child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseNode(c)
		}
	}

	parseNode(doc)
	return m
}

// resolveURL converts a relative URL to an absolute URL using the base URL.
func resolveURL(relativeURL, baseURL string) string {
	if relativeURL == "" {
		return ""
	}

	// Already absolute
	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return relativeURL
	}

	parsed, err := url.Parse(relativeURL)
	if err != nil {
		return relativeURL
	}

	// Handle protocol-relative URLs
	if strings.HasPrefix(relativeURL, "//") {
		return base.Scheme + ":" + relativeURL
	}

	return base.ResolveReference(parsed).String()
}

// extractFavicon extracts the favicon URL from the page.
func extractFavicon(baseURL string, body []byte) string {
	// First try to find favicon.ico in standard location
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	faviconURL := fmt.Sprintf("%s://%s/favicon.ico", base.Scheme, base.Host)

	// Look for icon link in HTML
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return faviconURL
	}

	var foundFavicon string
	var parseFaviconNode func(*html.Node)
	parseFaviconNode = func(n *html.Node) {
		if foundFavicon != "" {
			return
		}
		if n.Type == html.ElementNode {
			if n.Data == "link" {
				rel := ""
				href := ""
				for _, attr := range n.Attr {
					if attr.Key == "rel" {
						rel = strings.ToLower(attr.Val)
					}
					if attr.Key == "href" {
						href = attr.Val
					}
				}

				// Check for various favicon rel values
				if strings.Contains(rel, "icon") || strings.Contains(rel, "shortcut") {
					foundFavicon = resolveURL(href, baseURL)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseFaviconNode(c)
		}
	}

	parseFaviconNode(doc)

	if foundFavicon != "" {
		return foundFavicon
	}

	return faviconURL
}
