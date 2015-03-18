package server

import (
	"fmt"
	"reflect"

	"github.com/b2aio/typhon/errors"
	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
)

type Endpoint struct {
	Name     string
	Handler  func(Request) (proto.Message, error)
	Request  interface{}
	Response interface{}
}

func (e *Endpoint) HandleRequest(req Request) (proto.Message, error) {

	// @todo check that `Request` and `Response` are set in RegisterEndpoint
	// @todo don't tightly couple `HandleRequest` to the proto encoding

	if e.Request != nil {
		body := cloneTypedPtr(e.Request).(proto.Message)
		if err := proto.Unmarshal(req.Payload(), body); err != nil {
			return nil, errors.Wrap(fmt.Errorf("Could not unmarshal request"))
		}
		req.SetBody(body)
	}

	log.Debugf("%s.%s handler received request: %+v", req.Service(), e.Name, req.Body())

	resp, err := e.Handler(req)

	if err != nil {
		err = enrichError(err, req, e)
		log.Errorf("%s.%s handler error: %s", req.Service(), e.Name, err.Error())
	} else {
		log.Debugf("%s.%s handler response: %+v", req.Service(), e.Name, resp)
	}

	return resp, err
	// @todo return error if e.Response is set and doesn't match
}

// cloneTypedPtr takes a pointer of any type and returns a pointer to
// to a newly allocated instance of that same type.
// This allows us to write generic unmarshalling code that is independent
// of a endpoint's expected message type. This way we can handle
// unmarshalling errors outside of the actual endpoint handler methods.
func cloneTypedPtr(reqType interface{}) interface{} {
	// http://play.golang.org/p/MJOc3g7t23
	// `reflect.New` gives us a `reflect.Value`, using type of `reflect.TypeIf(e.RequestType()).Elem()` (a struct type)
	// and that struct's zero value for a value.
	// `reflectValue.Interface()` puts the type and value back together into an interface type
	reflectValue := reflect.New(reflect.TypeOf(reqType).Elem())
	return reflectValue.Interface()
}

// enrichError converts an error interface into *errors.Error and attaches
// lots of information to it.
// NOTE: if the error came from somewhere down the stack, it isn't modified
// @todo once the server context gives us a parent request and trace id, we can store even more information in the error!
func enrichError(err error, ctx Request, endpoint *Endpoint) *errors.Error {
	wrappedErr := errors.Wrap(err)

	// @todo an error will probably have a source_request_id or something that we can use to
	// more reliably make sure this information is only attached once, as the error travels up the service stack
	if wrappedErr.PrivateContext["service"] == "" {
		wrappedErr.PrivateContext["service"] = ctx.Service()
	}
	if wrappedErr.PrivateContext["endpoint"] == "" {
		wrappedErr.PrivateContext["endpoint"] = endpoint.Name
	}
	return wrappedErr
}
