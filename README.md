# beans

Dependency Injection Runtime Framework
Inspired by Spring Framework in Java

All injections happens on runtime and took O(n*m) complexity, where n - number of interfaces, m - number of services.
In golang we have to check each interface with each instance to know if they are compatible. 
All injectable fields must have tag `inject` and be public.

### Usage

SpringFramework-like golang DI framework.

Example:
```

type storageService struct {
    Logger *zap.Logger  `inject`
}

type userService struct {
	Storage app.Storage  `inject`
    Logger *zap.Logger  `inject`
}

type configService struct {
	Storage app.Storage  `inject`
    Logger *zap.Logger  `inject`
}

type appService struct {
    beans.InitializingBean
	beans.DisposableBean
	Context beans.Context  `inject`
}

// context.InitializingBean
func (t *appService) PostConstruct() error {
    // all fields are injected and dependency beans are constructed before this call
	return nil
}

// context.DisposableBean
func (t *appService) Destroy() error {
    // called on close context
	return nil
}

logger, _ := newLogger()
ctx, err := beans.Create(
		logger,
		storage,
		&configService{},
		&userService{},
        &appService{},
		&struct{
			UserService `inject`
			AppService  `inject`
		}{},
)
require.NoError(t, err)
defer ctx.Close()

beans := ctx.Bean("app.UserService")
b, ok := ctx.Bean(reflect.TypeOf((*app.UserService)(nil)).Elem())
```

### Factory Bean

Added support for Factory Beans, that could be singleton or not.

### Lazy fields

Added support for lazy fields, that defined like func() BeanType `inject`
