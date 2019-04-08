package offline

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/ortuman/jackal/xmpp"
	"github.com/sony/gobreaker"
)

type gateway interface {
	Route(msg *xmpp.Message) error
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type httpGateway struct {
	url       string
	authToken string
	reqBuf    *bytes.Buffer
	cb        *gobreaker.CircuitBreaker
	client    httpClient
}

func newHTTPGateway(url string, authToken string) gateway {
	return &httpGateway{
		url:       url,
		authToken: authToken,
		reqBuf:    bytes.NewBuffer(nil),
		cb:        gobreaker.NewCircuitBreaker(gobreaker.Settings{}),
		client:    &http.Client{},
	}
}

func (g *httpGateway) Route(msg *xmpp.Message) error {
	msg.ToXML(g.reqBuf, true)
	defer g.reqBuf.Reset()

	req, err := http.NewRequest(http.MethodPost, g.url, g.reqBuf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Authorization", g.authToken)

	_, err = g.cb.Execute(func() (i interface{}, e error) {
		resp, err := g.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("response status code: %d", resp.StatusCode)
		}
		return nil, nil
	})
	return err
}
