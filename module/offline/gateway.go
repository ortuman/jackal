package offline

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/ortuman/jackal/xmpp"
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
	client    httpClient
	reqBuf    *bytes.Buffer
}

func newHTTPGateway(url string, authToken string) gateway {
	return &httpGateway{
		url:       url,
		authToken: authToken,
		reqBuf:    bytes.NewBuffer(nil),
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

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response status code: %d", resp.StatusCode)
	}
	return nil
}
