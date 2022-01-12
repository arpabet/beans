# beans

Dependency Injection Runtime Framework for Golang inspired by Spring Framework in Java.

All injections happens on runtime and took O(n*m) complexity, where n - number of interfaces, m - number of services.
In golang we have to check each interface with each instance to know if they are compatible. 
All injectable fields must have tag `inject` and be public.

### Usage

Dependency Injection framework for complex applications written in Golang.
There is not capability to scan components in packages in Golang, therefore the context creation needs to accept all beans as created instances.

Example:
```
var ctx, err = beans.Create(
    logger,
    &storageImpl{},
    &configServiceImpl{},
    &userServiceImpl{},
    &struct {
        UserService UserService `inject`
    }{}, // could be used by runtime injects
)
require.Nil(t, err)
defer ctx.Close()
```

### Types

Bean Framework supports following types for beans:
* Pointer to Struct
* Interface
* Function

Beans Framework does not support Struct type as bean instance type. 

### Function

Function in golang is the first type citizen, therefore Bean Framework supports injection of functions by default.
All primitive types and non-bean collections recommended to inject by functions.

Example:
```
type holder struct {
	StringArray   func() []string `inject`
}

var ctx, err = beans.Create (
    &holder{},
    func() []string { return []string {"a", "b"} },
)

ctx.Close()
``` 
 
### Collections 
 
Beans Framework supports injection of bean collections like Slice and Map.

Example:
```
type holderImpl struct {
	Array   []Element          `inject`
	Map     map[string]Element `inject`
}

var ElementClass = reflect.TypeOf((*Element)(nil)).Elem()
type Element interface {
    beans.NamedBean
    beans.OrderedBean
}
```  
 
In this case Element can implement beans.NamedBean interface to override bean name and also implement beans.OrderedBean to assign order for the bean in collection.
 
### beans.InitializingBean

Added support for InitializingBean interface, whereas Beans Framework invokes PostConstruct method for each matching bean after injection phase.

Example:
```
type component struct {
    Dependency  *anotherComponent  `inject`
}

func (t *component) PostConstruct() error {
    if t.Dependency == nil {
        // for normal required dependency can not be happened, unless `optional` field declared
        return errors.New("empty dependency")
    }
    if !t.Dependency.Initialized() {
        // for normal required dependency can not be happened, unless `lazy` field declared
        return errors.New("not initialized dependency")
    }
    return t.Dependency.DoSomething()
}
``` 

### beans.DisposableBean

Added support for DisposableBean interface, whereas Beans Framework invokes Destroy method for each matching bean during Close context call in backwards initialization order.

Example:
```
type component struct {
    Dependency  *anotherComponent  `inject`
}

func (t *component) Destroy() error {
    // guarantees that dependency still not destroyed by calling in backwards initialization order
    return t.Dependency.DoSomething()
}
```

### beans.NamedBean

Added support for NamedBean interface to assign name to bean instance, used for qualifier bean injection.

Example:
```
type component struct {
}

func (t *component) BeanName() string {
    // overrides default bean name: package_name.component
    return "c"
}
```

### beans.OrderedBean

Added support for OrderedBean interface to inject beans with specific order. 
If bean does not implement OrderedBean interface, then Beans Framework preserve context initialization order. 

Example:
```
type component struct {
}

func (t *component) BeanOrder() int {
    // created ordered bean with order 100, would be injected in Slice(Array) in this order. 
    // first comes ordered beans, rest unordered with preserved order of initialization sequence.
    return 100
}
```

### beans.FactoryBean

Added support for FactoryBean interface, that used to create bean by application with required dependencies.
FactoryBean can produce singleton and non-singleton beans.

Example:
```
var beanConstructedClass = reflect.TypeOf((*beanConstructed)(nil))
type beanConstructed struct {
}

type factory struct {
    Dependency  *anotherComponent  `inject`
}

func (t *factory) Object() (interface{}, error) {
    if err := t.Dependency.DoSomething(); err != nil {
        return nil, err
    }
	return &beanConstructed{}, nil
}

func (t *factory) ObjectType() reflect.Type {
	return beanConstructedClass
}

func (t *factory) Singleton() bool {
	return true
}
```

### Lazy fields

Added support for lazy fields, that defined like this: `inject:"lazy"`.

Example:
```
type component struct {
    Dependency  *anotherComponent  `inject:"lazy"`
    Initialized bool
}

type anotherComponent struct {
    Dependency  *component  `inject`
    Initialized bool
}

func (t *component) PostConstruct() error {
    // all injected required fields can not be nil
    // but for lazy fields, to avoid cycle dependencies, the dependency field would be not initialized
    println(t.Dependency.Initialized) // output is false
    t.Initialized = true
}

func (t *anotherComponent) PostConstruct() error {
    // all injected required fields can not be nil and non-lazy dependency fields would be initialized
    println(t.Dependency.Initialized) // output is true
    t.Initialized = true
}
```

### Optional fields

Added support for optional fields, that defined like this: `inject:"optional"`.

Example:

Example:
```
type component struct {
    Dependency  *anotherComponent  `inject:"optional"`
}
```

Suppose we do not have anotherComponent in context, but would like our context to be created anyway, that is good for libraries.
In this case there is a high risk of having null-pointer panic during runtime, therefore for optional dependency
fields you need always check if it is not nil before use.

```
if t.Dependency != nil {
    t.Dependency.DoSomething()
}
```

