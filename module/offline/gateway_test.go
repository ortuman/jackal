package offline

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

type fakeHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (c *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.do(req)
}

func TestHttpGateway_Route(t *testing.T) {
	g := newHTTPGateway("http://127.0.0.1:6666").(*httpGateway)
	fakeClient := &fakeHTTPClient{}
	g.client = fakeClient

	msg := xmpp.NewMessageType(uuid.New().String(), xmpp.ChatType)
	body := xmpp.NewElementName("body")
	body.SetText("This is an offline message!")
	msg.AppendElement(body)

	var reqBody string
	fakeClient.do = func(req *http.Request) (response *http.Response, e error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "application/xml", req.Header.Get("Content-Type"))

		b, _ := ioutil.ReadAll(req.Body)
		reqBody = string(b)
		return &http.Response{StatusCode: http.StatusOK}, nil
	}

	err := g.Route(msg)
	require.Nil(t, err)
	require.Equal(t, msg.String(), reqBody)

	fakeClient.do = func(req *http.Request) (response *http.Response, e error) {
		return &http.Response{StatusCode: http.StatusInternalServerError}, nil
	}
	require.NotNil(t, g.Route(msg))

	fakeClient.do = func(req *http.Request) (response *http.Response, e error) {
		return nil, errors.New("foo error")
	}
	require.NotNil(t, g.Route(msg))
}
