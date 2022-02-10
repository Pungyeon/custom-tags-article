# Creating your own custom tags

## Introduction
> NOTE: You can [skip](#Retrieving tags) this if you already know what a struct tag is/does

What are struct tags? Struct tags is a Go feature, which defines a tag for a particular field of a struct. In itself, a struct tag is merely metadata and does not affect anything. However, it is commonly used for defining behaviour for that particular field. A very common example of this is the `json` tag, which the `json` package uses to Marshal and Unmarshal struct fields:

```go
type Person struct {
	Email string `json:"email"`
}
```

The above example instructs the `json` package to write the `Person::Email` field as `email` when encoding to JSON, rather than `Email`, as well as decoding the value of `email` field in a JSON and assigning it to `Person::Email`. Simple! This is really useful, because it's a very concise way of specifying this, which is the general idea of using struct tags.

However, you can use the struct tag for many things. This article will explain how to create your own custom tag and (hopefully) save yourself from writing the same code over and over again!

## Retrieving tags
The first hurdle we need to tackle, is figuring out how we can retrieve these tags programmatically. Fortunately,
retrieving a tag, is pretty straight forward, using the `reflect` package. As with any article which mentions the `reflect` package, a warning must ensue. The `reflect` package is a powerful package and gives Go developers the flexibility to create some very useful and creative projects. However, one must proceed with caution! The `reflect` package is unforgiving and errors are handled with a `panic`. We will see examples of this later in the article.

However, with that out of the way, here is how to retrieve struct tags using the `reflect` package:

```go
func PrintTags(v interface{}) {
	val := reflect.ValueOf(v)
	kind := val.Kind()
	switch kind {
	case reflect.Struct:
		typ := val.Type()
		for i := 0; i < typ.NumField(); i++ {
			fmt.Println(typ.Field(i).Tag)
		}
		return
	}
}
```

In the above function, we have created a function which accepts an `interface{}` value (in other words, this can be *any* value). We use the `reflect.ValueOf` function, to retrieve our `reflect.Value`.

```go
type Value struct {
	typ *rtype
	ptr unsafe.Pointer
	flag
}
```

Simply, the `typ` field of type `*rtype` is Go's "common" internal library type. It is essentially just metadata for that particular value: name, size, equality function, hash and garbage collection data. The `ptr unsafe.Pointer` is a raw pointer to the actual data of the value and `flag` is additional metadata: whether the value is read-only, if it is addressable etc. I'm not going deeper into this rabbit hole, but if are curious as to how the Go runtime works, I can highly recommend jumping in.

> NOTE : Thoroughly recommend this article series: https://cmc.gitbook.io/go-internals/

Either way, the `reflect.Value` type allows us to have a peak at some of the metadata of the given value. For example, using the method `reflect.Value::Kind` we can retrieve the underlying type (int, array, slice, struct etc.). Using this kind value, we can check whether the given value is of type `reflect.Struct`. We do this, as we are not interested in anything else; After all, we are trying to retrieve struct tags, and they only reside on structs.

Should we have been lucky enough to receive a struct kind, we will now retrieve the type information of this struct. As an example, if we had received a `Person` type, we would be retrieving the `reflect.Type` metadata for a `Person`. With this type information we can now iterate over the fields by calling the `reflect.Type::NumField` method, which will return the number of fields for that type. Thereafter, we can retrieve the metadata for each field using the method `reflect.Value::Field`, specifying the field index with our iterator `i`.

Last, but not least, we can now access the `reflect.Field::Tag` property, which is indeed the struct tag for that particular field. So, let's take it for a spin:


```go
type Person struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

func main() {
	PrintTags(Person{
		FirstName: "Lasse Martin",
		LastName:  "Jakobsen",
		Email:     "lasse@tengen.dk",
	})
}

Output:
json:"first_name"
json:"last_name"
json:"email"
```

This is great! We can already feel the power of the `reflect` package >:) Our newly created `Person` type has three fields, which are all being printed as expected. However, there is still work to be done, our current function is not particular successful at printing tags of inner struct fields:

```go
type Person struct {
	Name      Name   `json:"name"`
	Email     string `json:"email"`
}

type Name struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func main() {
	PrintTags(Person{
		Name: Name{
			FirstName: "Lasse Martin",
			LastName:  "Jakobsen",
		},
		Email:     "lasse@tengen.dk",
	})
}

Output:
json:"name"
json:"email"
```

As we can see above, we have moved our first and last name fields into a struct of their own `Name`. Our current functionality only looks at the immediate struct tags. We need to recursively check our fields, if they could themselves contain struct tags:

```go
func PrintTags(v interface{}) {
	val := reflect.ValueOf(v)
	kind := val.Kind()
	switch kind {
	case reflect.Struct:
		handleStruct(val)
	}
}

func handleStruct(val reflect.Value) {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fmt.Println(field.Tag)
		switch val.Field(i).Kind() {
		case reflect.Struct:
			handleStruct(val.Field(i))
		}
	}
	return
}
```

We have simply moved our logic into another function `handleStruct` which in turn checks if one of the fields of the given struct, is a struct itself. If so, then we simple call `handleStruct` again. Easy peasy! Running our `main` function again, will yield all of the tags of the inner struct :thumbs_up: - However, we also need to think about other kinds than inner structs; We also need to think about structs containing arrays, maps etc, which in turn, could also contain structs. However, this should be fairly simple to handle, as we can just add a few more handlers for our various types.

```go
func handleValue(val reflect.Value) {
	kind := val.Kind()
	switch kind {
	case reflect.Struct:
		handleStruct(val)
	case reflect.Array, reflect.Slice:
		handleArray(val)
	case reflect.Map:
		handleMap(val)
	case reflect.Ptr:
		handleValue(val.Elem())
	}
}

func handleStruct(val reflect.Value) {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		handleValue(val.Field(i))
	}
	return
}

func handleArray(val reflect.Value) {
	for i := 0; i < val.Len(); i++ {
		handleValue(val.Index(i))
	}
}

func handleMap(val reflect.Value) {
	for _, key := range val.MapKeys() {
		handleValue(val.MapIndex(key))
	}
}
```

Great, as can be seen above. We have added three new handlers, so that we are now handling structs, arrays/slices and maps. Each of them have a slightly difference syntax for iterating through their contents. For arrays and slices, we are using the `reflect.Value::Len` method to retrieve the length of the array and `reflect.Value::Index` for retrieving the element at the specified index. For maps we are iterating through the keys of the map and retrieving the value stored in that key.

It's important to note, that the `reflect.Value::NumField` and `reflect.Value::MapKeys` methods are specific to, respectively, structs and maps. If these methods are called on a different value kind, it will cause a panic, which we want to avoid at all costs.

> NOTE : We have also added a handler for pointers in `handleValue`. This is because `reflect` will identify a pointer as a `reflect.Ptr` rather than a struct (which makes sense). So, calling the `.Elem()`, essentially is the same as de-referencing, returning the value of that pointer.

Furthermore, we have also added `handleValue` which acts as a distributor, identifying the kind of the value and invoking the corresponding function for that kind.

## Creating Custom Tags
### The building blocks
So far so good, but currently we are only accessing the field tag, we aren't actually doing anything with them. But it's almost time. Before we move on to this step, let's do a super quick refactor:

```go
type TagHandler struct {
	HandlerFn func(value reflect.Value, field reflect.StructField) error
}

func (th TagHandler) Handle(v interface{}) error {
	return th.handleValue(reflect.ValueOf(v))
}


func (th TagHandler) handleStruct(val reflect.Value) error {
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		if err := th.HandlerFn(val.Field(i), typ.Field(i)); err != nil {
			return err
		}
		if err := th.handleValue(val.Field(i)); err != nil {
			return err
		}
	}
	return nil
}

func (th TagHandler) handleValue(val reflect.Value) error { ... }

func (th TagHandler) handleArray(val reflect.Value) error { ... }

func (th TagHandler) handleMap(val reflect.Value) error { ... }
```

We have created a new structure `TagHandler` and have made all of our functions into methods of this struct. Furthermore, `TagHandler` stores a function pointer with the signature `func(reflect.Value, reflect.StructField) error`, the idea behind this, is to allow any function with this signature to be called by the `TagHandler::handleStruct` method. This enables us to, very easily, create functionality for our custom tags. So let's try it out!

### Regex Validator Tag
We are going to create a custom tag, which will be able to validate the value of a tagged field, using a regular expression.

```go
func handleValidateTag(value reflect.Value, field reflect.StructField) error {
	tag, ok := field.Tag.Lookup("validate")
	if !ok {
		return nil
	}
	match, err := regexp.Compile(tag)
	if err != nil {
		return fmt.Errorf("validation regexp syntax error: %v", err)
	}
	if !match.MatchString(value.String()) {
		return fmt.Errorf("invalid field (%v::%v) %v != %v", field.Type, field.Name, value.String(), tag)
	}
	return nil
}
```

The function `handleValidateTag` receives a `reflect.Value` and a `reflect.StructField`. Using the struct field, we lookup the value for the tag `validate`. If it doesn't exist `!ok`, then we know that there is no `validate` tag and therefore nothing to validate, so we can safely just return. However, if there is a tag, we attempt to compile it and then match the field value with our tag regular expression. If there is no match, then the value is considered invalid, so we return an error. If there is a match, we can assume that the value is valid. Let's try it out!

```go
type Person struct {
	...
	Email string `json:"email" validate:"^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$"`
	...
}

func main() {
	th := TagHandler{
		HandlerFn: handleValidateTag,
	}

	err := th.Handle(Person{
		Name: Name{
			FirstName: "Lasse Martin",
			LastName:  "Jakobsen",
		},
		Email:     "lasse@tengen.dk",
		Friends: []*Person{
			{
				Name: Name{ FirstName: "Iaf", LastName: "Nofrens"},
				Email: "l33tboi95@hotmail",
			},
		},
	})
	fmt.Println(err)
}
```

> NOTE : The regex value for validating an e-mail is not 100% safe, but it should suffice for the purposes for this article :relaxed_smile:

Notice that this will return an error, because the e-mail in the friends slice is invalid. If we fix this email address by giving it a `.com` postfix, the error is resolved ! Magic ! :party:

Of course, our `handleValidateTag` is still a rather naive function. For example, it assumes that all fields will be of string value. This is an issue! It is easily imaginable that we wanted to validate something else, such as an integer. Let's try to add a `BirthYear` integer field to our `Person` type and see what happens, when we run our program.

```go
type Person struct {
	BirthYear int `json:"birth_year" validate:"^(19|20)\\d\\d$"`
	...
}

Output:
invalid field (int::BirthYear) int Value> != ^(19|20)\d\d$
```

So, this is because of the following line of code:
```go
func handleValidateTag(value reflect.Value, field reflect.StructField) error {
	...
	if !match.MatchString(value.String()) { ... }
	...
}
```

We are trying to access the string value of our `reflect.Value` using the method `reflect.Value::String`. However, in this case, our underlying value is actually an integer, so `reflect` returns the string value "int Value". So, thankfully not a panic, but nevertheless, completely useless. We will handle this lazily, but effectively but converting our type to string with `fmt.Sprintf` rather than using `reflect.Value::String`

```go
func valueToString(value reflect.Value) string {
	return fmt.Sprintf("%v", value.Interface())
}

func handleValidateTag(value reflect.Value, field reflect.StructField) error {
	...
	str := valueToString(value)
	if !match.MatchString(str) { ... }
	...
}
```

We have created a new function `valueToString` which uses `fmt.Sprintf` to return a string from the underlying `interface{}` contained in the `reflect.Value`. This is probably not the most efficient way of doing this, but it certainly does the job. If we run our program again, we will get the following output:

```
invalid field (int::BirthYear) 0 != ^(19|20)\d\d$
```

And if we change the value of `BirthYear` to something valid (within this century), our validator will stop complaining :party: There are of course many other cases we are not accounting for, but for now, we will put our validator on the shelf and move on to something else.

### Config Tag
So, now that we have seen that we can validate our struct field values via. our tags, how about we have a look at using struct tags for *setting* the values of our struct fields? Let's try to make a struct tag, in which we can specify the environment variable which should populate the value of our config parameter. This is a pretty common use-case and something that has been done many times before, but let's try doing this ourselves, to see what it involves.

Firstly, let's have a look at the syntax to use for specifying our environment variable parameters. I suggest that we start of simple, specifying only the name of our environment variable holding the value, so our config struct would look something like the following:

```go
type Config struct {
	HttpMaxRetries    int    `conf:"HTTP_MAX_RETRIES"`
	ElasticsearchHost string `conf:"ELASTICSEARCH_HOST"`
}
```

----------------------
Laziness
----------------------

Then we create our function for reading in our tags and setting the struct field value:

```go
func handleConfigTag(value reflect.Value, field reflect.StructField) error {
	tag, ok := field.Tag.Lookup("conf")
	if !ok {
		return nil
	}
	envvar, ok := os.LookupEnv(tag)
	if !ok {
		return nil
	}
	return setValue(value, envvar)
}

func setValue(value reflect.Value, envvar string) error {
	switch value.Kind() {
	case reflect.String:
		value.Set(reflect.ValueOf(envvar))
	case reflect.Int:
		n, err := strconv.Atoi(envvar)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(n))
	}
	return nil
}
```

The above example is doing the thing. We simply read the tag value and then lookup the environment variable specified.

Then we attempt to set the field value, by determining the field kind and then converting the string to the appropriate type.

Currently, we are just support `int` and `string`, but it won't take much to add support for other types. If we wanted to, we could go as far as adding support for slices, structs etc. ... However, we won't go that far in this article :sweat_smile:

Instead, let us simple test out our simple new configuration, to see if it works!

```go
func main() {
	cfgHandler := TagHandler{
		HandlerFn: handleConfigTag,
	}

	var cfg Config
	err = cfgHandler.Handle(&cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf(`ElasticsearchHost: %s, HttpMaxRetries: %d\n`,
		cfg.ElasticsearchHost, cfg.HttpMaxRetries)
}
```

Running

```bash
> ELASTICSEARCH_HOST=http://localhost:9200 HTTP_MAX_RETRIES=5 go run main.go
ElasticsearchHost: http://localhost:9200, HttpMaxRetries: 5
```

## Summary
We made it a thing, it was good.