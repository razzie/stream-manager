package client

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/razzie/geoip-server/geoip"
)

// Client is a lightweight http client to request location data from geoip-server
type Client struct {
	ServerAddress string
	mtx           sync.Mutex
	cache         map[string]*geoip.Location
}

// DefaultClient is the default client
var DefaultClient geoip.Client = NewClient("https://geoip.gorzsony.com")

// NewClient returns a new client
func NewClient(serverAddr string) *Client {
	return &Client{
		ServerAddress: serverAddr,
		cache:         make(map[string]*geoip.Location),
	}
}

// Provider returns the provider this client is requesting locations from
func (c *Client) Provider() string {
	u, err := url.Parse(c.ServerAddress)
	if err != nil {
		return c.ServerAddress
	}
	return u.Host
}

func (c *Client) getCachedLocation(hostname string) (*geoip.Location, bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	loc, ok := c.cache[hostname]
	return loc, ok
}

func (c *Client) cacheLocation(hostname string, loc *geoip.Location) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if _, ok := c.cache[hostname]; !ok {
		c.cache[hostname] = loc
		go func() {
			<-time.NewTimer(time.Minute * 5).C
			c.mtx.Lock()
			defer c.mtx.Unlock()
			delete(c.cache, hostname)
		}()
	}
}

// GetLocation retrieves the location data of an IP or hostname from geoip-server
func (c *Client) GetLocation(ctx context.Context, hostname string) (*geoip.Location, error) {
	if loc, ok := c.getCachedLocation(hostname); ok {
		return loc, nil
	}

	req, _ := http.NewRequest("GET", c.ServerAddress+"/"+hostname, nil)
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var loc geoip.Location
	err = json.Unmarshal(result, &loc)
	if err != nil {
		return nil, err
	}

	c.cacheLocation(hostname, &loc)
	return &loc, nil
}
