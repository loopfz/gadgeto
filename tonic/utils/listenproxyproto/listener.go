package listenproxyproto

import (
	"github.com/loopfz/gadgeto/tonic"
	"github.com/pires/go-proxyproto"
)

func ListenProxyProtocol(o *tonic.ListenOpt) error {
	o.Listener = &proxyproto.Listener{Listener: o.Listener}
	return nil
}
