package servers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Geo holds resolved geolocation for an IP.
type Geo struct {
	Country string
	City    string
	ASN     string
}

// Geolocator resolves geolocation for an IP address.
type Geolocator interface {
	Lookup(ctx context.Context, ip string) (Geo, error)
}

// IPAPIGeolocator resolves geo via the free ip-api.com endpoint. Failures are
// the caller's concern; the check flow treats geo as best-effort.
type IPAPIGeolocator struct {
	Client  *http.Client
	BaseURL string
}

// NewIPAPIGeolocator builds a geolocator with sensible defaults.
func NewIPAPIGeolocator() *IPAPIGeolocator {
	return &IPAPIGeolocator{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: "http://ip-api.com/json",
	}
}

// Lookup implements Geolocator.
func (g *IPAPIGeolocator) Lookup(ctx context.Context, ip string) (Geo, error) {
	url := fmt.Sprintf("%s/%s?fields=status,country,city,as", g.BaseURL, ip)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Geo{}, err
	}
	resp, err := g.Client.Do(req)
	if err != nil {
		return Geo{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	var body struct {
		Status  string `json:"status"`
		Country string `json:"country"`
		City    string `json:"city"`
		AS      string `json:"as"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Geo{}, err
	}
	if body.Status != "success" {
		return Geo{}, fmt.Errorf("geo lookup failed for %s", ip)
	}
	return Geo{Country: body.Country, City: body.City, ASN: body.AS}, nil
}
