// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package fuse

import (
	"net/http"
	"sync"
)

var (
	lockhttpRequesterMockDo sync.RWMutex
)

// httpRequesterMock is a mock implementation of httpRequester.
//
//     func TestSomethingThatUseshttpRequester(t *testing.T) {
//
//         // make and configure a mocked httpRequester
//         mockedhttpRequester := &httpRequesterMock{
//             DoFunc: func(r *http.Request) (*http.Response, error) {
// 	               panic("TODO: mock out the Do method")
//             },
//         }
//
//         // TODO: use mockedhttpRequester in code that requires httpRequester
//         //       and then make assertions.
//
//     }
type httpRequesterMock struct {
	// DoFunc mocks the Do method.
	DoFunc func(r *http.Request) (*http.Response, error)

	// calls tracks calls to the methods.
	calls struct {
		// Do holds details about calls to the Do method.
		Do []struct {
			// R is the r argument value.
			R *http.Request
		}
	}
}

// Do calls DoFunc.
func (mock *httpRequesterMock) Do(r *http.Request) (*http.Response, error) {
	if mock.DoFunc == nil {
		panic("httpRequesterMock.DoFunc: method is nil but httpRequester.Do was just called")
	}
	callInfo := struct {
		R *http.Request
	}{
		R: r,
	}
	lockhttpRequesterMockDo.Lock()
	mock.calls.Do = append(mock.calls.Do, callInfo)
	lockhttpRequesterMockDo.Unlock()
	return mock.DoFunc(r)
}

// DoCalls gets all the calls that were made to Do.
// Check the length with:
//     len(mockedhttpRequester.DoCalls())
func (mock *httpRequesterMock) DoCalls() []struct {
	R *http.Request
} {
	var calls []struct {
		R *http.Request
	}
	lockhttpRequesterMockDo.RLock()
	calls = mock.calls.Do
	lockhttpRequesterMockDo.RUnlock()
	return calls
}
