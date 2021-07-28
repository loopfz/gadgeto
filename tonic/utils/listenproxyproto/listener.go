package listenproxyproto

import (
	"time"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/pires/go-proxyproto"
)

func ListenProxyProtocol(o *tonic.ListenOpt) error {
	o.Listener = &proxyproto.Listener{
		Listener:          o.Listener,
		ReadHeaderTimeout: 2 * time.Second,
	}
	return nil
}
