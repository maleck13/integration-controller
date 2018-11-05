// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package fuse

import (
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"sync"
)

var (
	lockCrudlerMockCreate sync.RWMutex
	lockCrudlerMockDelete sync.RWMutex
	lockCrudlerMockGet    sync.RWMutex
	lockCrudlerMockList   sync.RWMutex
	lockCrudlerMockUpdate sync.RWMutex
)

// CrudlerMock is a mock implementation of Crudler.
//
//     func TestSomethingThatUsesCrudler(t *testing.T) {
//
//         // make and configure a mocked Crudler
//         mockedCrudler := &CrudlerMock{
//             CreateFunc: func(object sdk.Object) error {
// 	               panic("TODO: mock out the Create method")
//             },
//             DeleteFunc: func(object sdk.Object) error {
// 	               panic("TODO: mock out the Delete method")
//             },
//             GetFunc: func(into sdk.Object, opts ...sdk.GetOption) error {
// 	               panic("TODO: mock out the Get method")
//             },
//             ListFunc: func(namespace string, o sdk.Object, option ...sdk.ListOption) error {
// 	               panic("TODO: mock out the List method")
//             },
//             UpdateFunc: func(object sdk.Object) error {
// 	               panic("TODO: mock out the Update method")
//             },
//         }
//
//         // TODO: use mockedCrudler in code that requires Crudler
//         //       and then make assertions.
//
//     }
type CrudlerMock struct {
	// CreateFunc mocks the Create method.
	CreateFunc func(object sdk.Object) error

	// DeleteFunc mocks the Delete method.
	DeleteFunc func(object sdk.Object) error

	// GetFunc mocks the Get method.
	GetFunc func(into sdk.Object, opts ...sdk.GetOption) error

	// ListFunc mocks the List method.
	ListFunc func(namespace string, o sdk.Object, option ...sdk.ListOption) error

	// UpdateFunc mocks the Update method.
	UpdateFunc func(object sdk.Object) error

	// calls tracks calls to the methods.
	calls struct {
		// Create holds details about calls to the Create method.
		Create []struct {
			// Object is the object argument value.
			Object sdk.Object
		}
		// Delete holds details about calls to the Delete method.
		Delete []struct {
			// Object is the object argument value.
			Object sdk.Object
		}
		// Get holds details about calls to the Get method.
		Get []struct {
			// Into is the into argument value.
			Into sdk.Object
			// Opts is the opts argument value.
			Opts []sdk.GetOption
		}
		// List holds details about calls to the List method.
		List []struct {
			// Namespace is the namespace argument value.
			Namespace string
			// O is the o argument value.
			O sdk.Object
			// Option is the option argument value.
			Option []sdk.ListOption
		}
		// Update holds details about calls to the Update method.
		Update []struct {
			// Object is the object argument value.
			Object sdk.Object
		}
	}
}

// Create calls CreateFunc.
func (mock *CrudlerMock) Create(object sdk.Object) error {
	if mock.CreateFunc == nil {
		panic("CrudlerMock.CreateFunc: method is nil but Crudler.Create was just called")
	}
	callInfo := struct {
		Object sdk.Object
	}{
		Object: object,
	}
	lockCrudlerMockCreate.Lock()
	mock.calls.Create = append(mock.calls.Create, callInfo)
	lockCrudlerMockCreate.Unlock()
	return mock.CreateFunc(object)
}

// CreateCalls gets all the calls that were made to Create.
// Check the length with:
//     len(mockedCrudler.CreateCalls())
func (mock *CrudlerMock) CreateCalls() []struct {
	Object sdk.Object
} {
	var calls []struct {
		Object sdk.Object
	}
	lockCrudlerMockCreate.RLock()
	calls = mock.calls.Create
	lockCrudlerMockCreate.RUnlock()
	return calls
}

// Delete calls DeleteFunc.
func (mock *CrudlerMock) Delete(object sdk.Object) error {
	if mock.DeleteFunc == nil {
		panic("CrudlerMock.DeleteFunc: method is nil but Crudler.Delete was just called")
	}
	callInfo := struct {
		Object sdk.Object
	}{
		Object: object,
	}
	lockCrudlerMockDelete.Lock()
	mock.calls.Delete = append(mock.calls.Delete, callInfo)
	lockCrudlerMockDelete.Unlock()
	return mock.DeleteFunc(object)
}

// DeleteCalls gets all the calls that were made to Delete.
// Check the length with:
//     len(mockedCrudler.DeleteCalls())
func (mock *CrudlerMock) DeleteCalls() []struct {
	Object sdk.Object
} {
	var calls []struct {
		Object sdk.Object
	}
	lockCrudlerMockDelete.RLock()
	calls = mock.calls.Delete
	lockCrudlerMockDelete.RUnlock()
	return calls
}

// Get calls GetFunc.
func (mock *CrudlerMock) Get(into sdk.Object, opts ...sdk.GetOption) error {
	if mock.GetFunc == nil {
		panic("CrudlerMock.GetFunc: method is nil but Crudler.Get was just called")
	}
	callInfo := struct {
		Into sdk.Object
		Opts []sdk.GetOption
	}{
		Into: into,
		Opts: opts,
	}
	lockCrudlerMockGet.Lock()
	mock.calls.Get = append(mock.calls.Get, callInfo)
	lockCrudlerMockGet.Unlock()
	return mock.GetFunc(into, opts...)
}

// GetCalls gets all the calls that were made to Get.
// Check the length with:
//     len(mockedCrudler.GetCalls())
func (mock *CrudlerMock) GetCalls() []struct {
	Into sdk.Object
	Opts []sdk.GetOption
} {
	var calls []struct {
		Into sdk.Object
		Opts []sdk.GetOption
	}
	lockCrudlerMockGet.RLock()
	calls = mock.calls.Get
	lockCrudlerMockGet.RUnlock()
	return calls
}

// List calls ListFunc.
func (mock *CrudlerMock) List(namespace string, o sdk.Object, option ...sdk.ListOption) error {
	if mock.ListFunc == nil {
		panic("CrudlerMock.ListFunc: method is nil but Crudler.List was just called")
	}
	callInfo := struct {
		Namespace string
		O         sdk.Object
		Option    []sdk.ListOption
	}{
		Namespace: namespace,
		O:         o,
		Option:    option,
	}
	lockCrudlerMockList.Lock()
	mock.calls.List = append(mock.calls.List, callInfo)
	lockCrudlerMockList.Unlock()
	return mock.ListFunc(namespace, o, option...)
}

// ListCalls gets all the calls that were made to List.
// Check the length with:
//     len(mockedCrudler.ListCalls())
func (mock *CrudlerMock) ListCalls() []struct {
	Namespace string
	O         sdk.Object
	Option    []sdk.ListOption
} {
	var calls []struct {
		Namespace string
		O         sdk.Object
		Option    []sdk.ListOption
	}
	lockCrudlerMockList.RLock()
	calls = mock.calls.List
	lockCrudlerMockList.RUnlock()
	return calls
}

// Update calls UpdateFunc.
func (mock *CrudlerMock) Update(object sdk.Object) error {
	if mock.UpdateFunc == nil {
		panic("CrudlerMock.UpdateFunc: method is nil but Crudler.Update was just called")
	}
	callInfo := struct {
		Object sdk.Object
	}{
		Object: object,
	}
	lockCrudlerMockUpdate.Lock()
	mock.calls.Update = append(mock.calls.Update, callInfo)
	lockCrudlerMockUpdate.Unlock()
	return mock.UpdateFunc(object)
}

// UpdateCalls gets all the calls that were made to Update.
// Check the length with:
//     len(mockedCrudler.UpdateCalls())
func (mock *CrudlerMock) UpdateCalls() []struct {
	Object sdk.Object
} {
	var calls []struct {
		Object sdk.Object
	}
	lockCrudlerMockUpdate.RLock()
	calls = mock.calls.Update
	lockCrudlerMockUpdate.RUnlock()
	return calls
}
