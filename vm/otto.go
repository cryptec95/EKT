package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/crypto"
	"github.com/EducationEKT/EKT/util"
	"github.com/EducationEKT/EKT/vm/file"
	"github.com/EducationEKT/EKT/vm/registry"
)

var (
	TIMEOUT_ERROR = errors.New("vm call error: out of time")
)

// Otto is the representation of the JavaScript runtime. Each instance of Otto has a self-contained namespace.
type Otto struct {
	// Interrupt is a channel for interrupting the runtime. You can use this to halt a long running execution, for example.
	// See "Halting Problem" for more information.
	Interrupt chan func()
	runtime   *_runtime
	rc        int // random count
}

type ContractData struct {
	contractData *types.ContractData
	err          error
}

func NewContractData(contractData *types.ContractData, err error) *ContractData {
	return &ContractData{
		contractData: contractData,
		err:          err,
	}
}

func (otto *Otto) UpgradeContract(code []byte, contractData *types.ContractData, timeout time.Duration) (*types.ContractData, error) {
	ch := make(chan *ContractData)
	go otto.upgradeContract(code, contractData, ch)
	for {
		select {
		case <-time.After(timeout):
			return nil, TIMEOUT_ERROR
		case result := <-ch:
			return result.contractData, result.err
		}
	}
}

func (otto *Otto) InitContractWithTimeout(code []byte, timeout time.Duration) (*types.ContractData, error) {
	ch := make(chan *ContractData)
	go otto.initContract(code, ch)
	for {
		select {
		case <-time.After(timeout):
			return nil, TIMEOUT_ERROR
		case result := <-ch:
			return result.contractData, result.err
		}
	}
}

func (otto *Otto) upgradeContract(code []byte, data *types.ContractData, ch chan *ContractData) {
	_, err := otto.Run(string(code))
	if err != nil {
		ch <- NewContractData(nil, err)
	}
	if otto.Set("propOld", data.Prop) != nil || otto.Set("contractOldStr", data.Contract) != nil {
		ch <- NewContractData(nil, err)
	}
	_, err = otto.Run(`
		init();
		var contractOld = JSON.parse(contractOldStr);

		upgrade();
	`)
	if err != nil {
		ch <- NewContractData(nil, err)
	}

	_, err = otto.Run(`
		var propStr = JSON.stringify(prop);
		var contractStr = JSON.stringify(contract);
		var contractData = JSON.stringify({ "prop": propStr, "contract": contractStr });
	`)
	if err != nil {
		ch <- NewContractData(nil, err)
	}

	value, err := otto.Get("contractData")
	if err != nil {
		ch <- NewContractData(nil, err)
	}
	newData := []byte(value.string())
	var contractData types.ContractData
	err = json.Unmarshal(newData, &contractData)
	ch <- NewContractData(&contractData, err)
}

func (otto *Otto) initContract(code []byte, ch chan *ContractData) {
	_, err := otto.Run(string(code))
	if err != nil {
		ch <- NewContractData(nil, err)
	}
	_, err = otto.Run(`
		init();
		var propStr = JSON.stringify(prop);
		var contractStr = JSON.stringify(contract);
		var contractData = JSON.stringify({ "prop": propStr, "contract": contractStr });
	`)
	value, err := otto.Get("contractData")
	if err != nil {
		ch <- NewContractData(nil, err)
	}
	data := []byte(value.string())
	var contractData types.ContractData
	err = json.Unmarshal(data, &contractData)
	ch <- NewContractData(&contractData, err)
}

func NewVM(lastHash []byte, timestamp int64) *Otto {
	self := &Otto{
		runtime: newContext(),
		rc:      0,
	}
	self.runtime.otto = self
	self.runtime.traceLimit = 10
	self.runtime.timestamp = timestamp
	self.Set("console", self.runtime.newConsole())

	self.SetRandomSource(func(vm *Otto) float64 {
		hash := crypto.Sha3_256(lastHash)
		rcHash := crypto.Sha3_256([]byte(strconv.Itoa(vm.rc)))
		seed1 := util.BytesToInt(hash)
		seed2 := util.BytesToInt(rcHash)
		seed := seed1 ^ seed2
		r := rand.New(rand.NewSource(int64(seed)))
		vm.rc++
		return r.Float64()
	})

	registry.Apply(func(entry registry.Entry) {
		self.Run(entry.Source())
	})

	return self
}

// New will allocate a new JavaScript runtime
func New() *Otto {
	self := &Otto{
		runtime: newContext(),
	}
	self.runtime.otto = self
	self.runtime.traceLimit = 10
	self.Set("console", self.runtime.newConsole())

	registry.Apply(func(entry registry.Entry) {
		self.Run(entry.Source())
	})

	return self
}

func (otto *Otto) clone() *Otto {
	self := &Otto{
		runtime: otto.runtime.clone(),
	}
	self.runtime.otto = self
	return self
}

// Run will allocate a new JavaScript runtime, run the given source
// on the allocated runtime, and return the runtime, resulting value, and
// error (if any).
//
// src may be a string, a byte slice, a bytes.Buffer, or an io.Reader, but it MUST always be in UTF-8.
//
// src may also be a Script.
//
// src may also be a Program, but if the AST has been modified, then runtime behavior is undefined.
//
func Run(src interface{}) (*Otto, Value, error) {
	otto := New()
	value, err := otto.Run(src) // This already does safety checking
	return otto, value, err
}

// Run will run the given source (parsing it first if necessary), returning the resulting value and error (if any)
//
// src may be a string, a byte slice, a bytes.Buffer, or an io.Reader, but it MUST always be in UTF-8.
//
// If the runtime is unable to parse source, then this function will return undefined and the parse error (nothing
// will be evaluated in this case).
//
// src may also be a Script.
//
// src may also be a Program, but if the AST has been modified, then runtime behavior is undefined.
//
func (self Otto) Run(src interface{}) (Value, error) {
	value, err := self.runtime.cmpl_run(src, nil)
	if !value.safe() {
		value = Value{}
	}
	return value, err
}

// Eval will do the same thing as Run, except without leaving the current scope.
//
// By staying in the same scope, the code evaluated has access to everything
// already defined in the current stack frame. This is most useful in, for
// example, a debugger call.
func (self Otto) Eval(src interface{}) (Value, error) {
	if self.runtime.scope == nil {
		self.runtime.enterGlobalScope()
		defer self.runtime.leaveScope()
	}

	value, err := self.runtime.cmpl_eval(src, nil)
	if !value.safe() {
		value = Value{}
	}
	return value, err
}

// Get the value of the top-level binding of the given name.
//
// If there is an error (like the binding does not exist), then the value
// will be undefined.
func (self Otto) Get(name string) (Value, error) {
	value := Value{}
	err := catchPanic(func() {
		value = self.getValue(name)
	})
	if !value.safe() {
		value = Value{}
	}
	return value, err
}

func (self Otto) getValue(name string) Value {
	return self.runtime.globalStash.getBinding(name, false)
}

// Set the top-level binding of the given name to the given value.
//
// Set will automatically apply ToValue to the given value in order
// to convert it to a JavaScript value (type Value).
//
// If there is an error (like the binding is read-only, or the ToValue conversion
// fails), then an error is returned.
//
// If the top-level binding does not exist, it will be created.
func (self Otto) Set(name string, value interface{}) error {
	{
		value, err := self.ToValue(value)
		if err != nil {
			return err
		}
		err = catchPanic(func() {
			self.setValue(name, value)
		})
		return err
	}
}

func (self Otto) setValue(name string, value Value) {
	self.runtime.globalStash.setValue(name, value, false)
}

func (self Otto) SetDebuggerHandler(fn func(vm *Otto)) {
	self.runtime.debugger = fn
}

func (self Otto) SetRandomSource(fn func(vm *Otto) float64) {
	self.runtime.random = fn
}

// SetStackDepthLimit sets an upper limit to the depth of the JavaScript
// stack. In simpler terms, this limits the number of "nested" function calls
// you can make in a particular interpreter instance.
//
// Note that this doesn't take into account the Go stack depth. If your
// JavaScript makes a call to a Go function, otto won't keep track of what
// happens outside the interpreter. So if your Go function is infinitely
// recursive, you're still in trouble.
func (self Otto) SetStackDepthLimit(limit int) {
	self.runtime.stackLimit = limit
}

// SetStackTraceLimit sets an upper limit to the number of stack frames that
// otto will use when formatting an error's stack trace. By default, the limit
// is 10. This is consistent with V8 and SpiderMonkey.
//
// TODO: expose via `Error.stackTraceLimit`
func (self Otto) SetStackTraceLimit(limit int) {
	self.runtime.traceLimit = limit
}

// MakeCustomError creates a new Error object with the given name and message,
// returning it as a Value.
func (self Otto) MakeCustomError(name, message string) Value {
	return self.runtime.toValue(self.runtime.newError(name, self.runtime.toValue(message), 0))
}

// MakeRangeError creates a new RangeError object with the given message,
// returning it as a Value.
func (self Otto) MakeRangeError(message string) Value {
	return self.runtime.toValue(self.runtime.newRangeError(self.runtime.toValue(message)))
}

// MakeSyntaxError creates a new SyntaxError object with the given message,
// returning it as a Value.
func (self Otto) MakeSyntaxError(message string) Value {
	return self.runtime.toValue(self.runtime.newSyntaxError(self.runtime.toValue(message)))
}

// MakeTypeError creates a new TypeError object with the given message,
// returning it as a Value.
func (self Otto) MakeTypeError(message string) Value {
	return self.runtime.toValue(self.runtime.newTypeError(self.runtime.toValue(message)))
}

// Context is a structure that contains information about the current execution
// context.
type Context struct {
	Filename   string
	Line       int
	Column     int
	Callee     string
	Symbols    map[string]Value
	This       Value
	Stacktrace []string
}

// Context returns the current execution context of the vm, traversing up to
// ten stack frames, and skipping any innermost native function stack frames.
func (self Otto) Context() Context {
	return self.ContextSkip(10, true)
}

// ContextLimit returns the current execution context of the vm, with a
// specific limit on the number of stack frames to traverse, skipping any
// innermost native function stack frames.
func (self Otto) ContextLimit(limit int) Context {
	return self.ContextSkip(limit, true)
}

// ContextSkip returns the current execution context of the vm, with a
// specific limit on the number of stack frames to traverse, optionally
// skipping any innermost native function stack frames.
func (self Otto) ContextSkip(limit int, skipNative bool) (ctx Context) {
	// Ensure we are operating in a scope
	if self.runtime.scope == nil {
		self.runtime.enterGlobalScope()
		defer self.runtime.leaveScope()
	}

	scope := self.runtime.scope
	frame := scope.frame

	for skipNative && frame.native && scope.outer != nil {
		scope = scope.outer
		frame = scope.frame
	}

	// Get location information
	ctx.Filename = "<unknown>"
	ctx.Callee = frame.callee

	switch {
	case frame.native:
		ctx.Filename = frame.nativeFile
		ctx.Line = frame.nativeLine
		ctx.Column = 0
	case frame.file != nil:
		ctx.Filename = "<anonymous>"

		if p := frame.file.Position(file.Idx(frame.offset)); p != nil {
			ctx.Line = p.Line
			ctx.Column = p.Column

			if p.Filename != "" {
				ctx.Filename = p.Filename
			}
		}
	}

	// Get the current scope this Value
	ctx.This = toValue_object(scope.this)

	// Build stacktrace (up to 10 levels deep)
	ctx.Symbols = make(map[string]Value)
	ctx.Stacktrace = append(ctx.Stacktrace, frame.location())
	for limit != 0 {
		// Get variables
		stash := scope.lexical
		for {
			for _, name := range getStashProperties(stash) {
				if _, ok := ctx.Symbols[name]; !ok {
					ctx.Symbols[name] = stash.getBinding(name, true)
				}
			}
			stash = stash.outer()
			if stash == nil || stash.outer() == nil {
				break
			}
		}

		scope = scope.outer
		if scope == nil {
			break
		}
		if scope.frame.offset >= 0 {
			ctx.Stacktrace = append(ctx.Stacktrace, scope.frame.location())
		}
		limit--
	}

	return
}

// Call the given JavaScript with a given this and arguments.
//
// If this is nil, then some special handling takes place to determine the proper
// this value, falling back to a "standard" invocation if necessary (where this is
// undefined).
//
// If source begins with "new " (A lowercase new followed by a space), then
// Call will invoke the function constructor rather than performing a function call.
// In this case, the this argument has no effect.
//
//      // value is a String object
//      value, _ := vm.Call("Object", nil, "Hello, World.")
//
//      // Likewise...
//      value, _ := vm.Call("new Object", nil, "Hello, World.")
//
//      // This will perform a concat on the given array and return the result
//      // value is [ 1, 2, 3, undefined, 4, 5, 6, 7, "abc" ]
//      value, _ := vm.Call(`[ 1, 2, 3, undefined, 4 ].concat`, nil, 5, 6, 7, "abc")
//
func (self Otto) Call(source string, this interface{}, argumentList ...interface{}) (Value, error) {

	thisValue := Value{}

	construct := false
	if strings.HasPrefix(source, "new ") {
		source = source[4:]
		construct = true
	}

	// FIXME enterGlobalScope
	self.runtime.enterGlobalScope()
	defer func() {
		self.runtime.leaveScope()
	}()

	if !construct && this == nil {
		program, err := self.runtime.cmpl_parse("", source+"()", nil)
		if err == nil {
			if node, ok := program.body[0].(*_nodeExpressionStatement); ok {
				if node, ok := node.expression.(*_nodeCallExpression); ok {
					var value Value
					err := catchPanic(func() {
						value = self.runtime.cmpl_evaluate_nodeCallExpression(node, argumentList)
					})
					if err != nil {
						return Value{}, err
					}
					return value, nil
				}
			}
		}
	} else {
		value, err := self.ToValue(this)
		if err != nil {
			return Value{}, err
		}
		thisValue = value
	}

	{
		this := thisValue

		fn, err := self.Run(source)
		if err != nil {
			return Value{}, err
		}

		if construct {
			result, err := fn.constructSafe(self.runtime, this, argumentList...)
			if err != nil {
				return Value{}, err
			}
			return result, nil
		}

		result, err := fn.Call(this, argumentList...)
		if err != nil {
			return Value{}, err
		}
		return result, nil
	}
}

// Object will run the given source and return the result as an object.
//
// For example, accessing an existing object:
//
//		object, _ := vm.Object(`Number`)
//
// Or, creating a new object:
//
//		object, _ := vm.Object(`({ xyzzy: "Nothing happens." })`)
//
// Or, creating and assigning an object:
//
//		object, _ := vm.Object(`xyzzy = {}`)
//		object.Set("volume", 11)
//
// If there is an error (like the source does not result in an object), then
// nil and an error is returned.
func (self Otto) Object(source string) (*Object, error) {
	value, err := self.runtime.cmpl_run(source, nil)
	if err != nil {
		return nil, err
	}
	if value.IsObject() {
		return value.Object(), nil
	}
	return nil, fmt.Errorf("value is not an object")
}

// ToValue will convert an interface{} value to a value digestible by otto/JavaScript.
func (self Otto) ToValue(value interface{}) (Value, error) {
	return self.runtime.safeToValue(value)
}

// Copy will create a copy/clone of the runtime.
//
// Copy is useful for saving some time when creating many similar runtimes.
//
// This method works by walking the original runtime and cloning each object, scope, stash,
// etc. into a new runtime.
//
// Be on the lookout for memory leaks or inadvertent sharing of resources.
func (in *Otto) Copy() *Otto {
	out := &Otto{
		runtime: in.runtime.clone(),
	}
	out.runtime.otto = out
	return out
}

// Object{}

// Object is the representation of a JavaScript object.
type Object struct {
	object *_object
	value  Value
}

func _newObject(object *_object, value Value) *Object {
	// value MUST contain object!
	return &Object{
		object: object,
		value:  value,
	}
}

// Call a method on the object.
//
// It is essentially equivalent to:
//
//		var method, _ := object.Get(name)
//		method.Call(object, argumentList...)
//
// An undefined value and an error will result if:
//
//		1. There is an error during conversion of the argument list
//		2. The property is not actually a function
//		3. An (uncaught) exception is thrown
//
func (self Object) Call(name string, argumentList ...interface{}) (Value, error) {
	// TODO: Insert an example using JavaScript below...
	// e.g., Object("JSON").Call("stringify", ...)

	function, err := self.Get(name)
	if err != nil {
		return Value{}, err
	}
	return function.Call(self.Value(), argumentList...)
}

// Value will return self as a value.
func (self Object) Value() Value {
	return self.value
}

// Get the value of the property with the given name.
func (self Object) Get(name string) (Value, error) {
	value := Value{}
	err := catchPanic(func() {
		value = self.object.get(name)
	})
	if !value.safe() {
		value = Value{}
	}
	return value, err
}

// Set the property of the given name to the given value.
//
// An error will result if the setting the property triggers an exception (i.e. read-only),
// or there is an error during conversion of the given value.
func (self Object) Set(name string, value interface{}) error {
	{
		value, err := self.object.runtime.safeToValue(value)
		if err != nil {
			return err
		}
		err = catchPanic(func() {
			self.object.put(name, value, true)
		})
		return err
	}
}

// Keys gets the keys for the given object.
//
// Equivalent to calling Object.keys on the object.
func (self Object) Keys() []string {
	var keys []string
	self.object.enumerate(false, func(name string) bool {
		keys = append(keys, name)
		return true
	})
	return keys
}

// KeysByParent gets the keys (and those of the parents) for the given object,
// in order of "closest" to "furthest".
func (self Object) KeysByParent() [][]string {
	var a [][]string

	for o := self.object; o != nil; o = o.prototype {
		var l []string

		o.enumerate(false, func(name string) bool {
			l = append(l, name)
			return true
		})

		a = append(a, l)
	}

	return a
}

// Class will return the class string of the object.
//
// The return value will (generally) be one of:
//
//		Object
//		Function
//		Array
//		String
//		Number
//		Boolean
//		Date
//		RegExp
//
func (self Object) Class() string {
	return self.object.class
}
